package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lostSetReplyConn discards the first SET reply after Redis has executed it.
// The shared state keeps the fault limited to one reply across reconnections.
type lostSetReplyConn struct {
	net.Conn
	dropped  *atomic.Bool
	writes   *atomic.Int32
	dropNext bool
}

func (c *lostSetReplyConn) Write(p []byte) (int, error) {
	if bytes.Contains(bytes.ToLower(p), []byte("\r\nset\r\n")) {
		c.writes.Add(1)
		c.dropNext = c.dropped.CompareAndSwap(false, true)
	}
	return c.Conn.Write(p)
}

func (c *lostSetReplyConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if c.dropNext && n > 0 {
		c.dropNext = false
		return 0, io.EOF
	}
	return n, err
}

func TestCreateShortURLLostReply(t *testing.T) {
	for _, requestedKey := range []string{"custom", ""} {
		t.Run(fmt.Sprintf("key=%q", requestedKey), func(t *testing.T) {
			server := miniredis.RunT(t)
			var dropped atomic.Bool
			var writes atomic.Int32
			client := redis.NewClient(&redis.Options{
				Addr: server.Addr(),
				// Keep the production client's default command retry policy.
				Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
					conn, err := new(net.Dialer).DialContext(ctx, network, addr)
					if err != nil {
						return nil, err
					}
					return &lostSetReplyConn{Conn: conn, dropped: &dropped, writes: &writes}, nil
				},
			})
			t.Cleanup(func() { client.Close() })
			creator := newTestCreator(t, client)
			generated := 0
			creator.generateKey = func() (string, error) {
				generated++
				return fmt.Sprintf("generated%d", generated), nil
			}

			key, err := creator.Create(context.Background(), "https://example.com", requestedKey)
			require.True(t, dropped.Load(), "the test must discard a SET reply")
			assert.ErrorIs(t, err, io.EOF, "a lost reply must remain a storage error")
			assert.Empty(t, key)
			assert.EqualValues(t, 1, writes.Load(), "do not resend an uncertain write")
			assert.Equal(t, 3, client.Options().MaxRetries, "preserve retries for other Redis operations")
			if requestedKey == "" {
				assert.Equal(t, 1, generated, "a storage failure must not generate a new key")
				assert.Equal(t, []string{"generated1"}, server.Keys())
			} else {
				assert.Zero(t, generated)
				assert.Equal(t, []string{requestedKey}, server.Keys())
			}
		})
	}
}
