package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lostReplyProxy forwards traffic to Redis, but closes the client connection in
// place of the first reply to the given command, after Redis has executed it.
// The shared state keeps the fault limited to one reply across reconnections.
type lostReplyProxy struct {
	listener net.Listener
	upstream string
	command  []byte
	dropped  atomic.Bool
	writes   atomic.Int32
}

func newLostReplyProxy(t *testing.T, upstream, command string) *lostReplyProxy {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })

	proxy := &lostReplyProxy{
		listener: listener,
		upstream: upstream,
		command:  []byte("\r\n" + command + "\r\n"),
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go proxy.forward(conn)
		}
	}()
	return proxy
}

func (p *lostReplyProxy) forward(client net.Conn) {
	defer client.Close()
	server, err := net.Dial("tcp", p.upstream)
	if err != nil {
		return
	}
	defer server.Close()

	var dropNext atomic.Bool
	go func() {
		defer server.Close()
		buf := make([]byte, 4096)
		for {
			n, err := client.Read(buf)
			if err != nil {
				return
			}
			if bytes.Contains(bytes.ToLower(buf[:n]), p.command) {
				p.writes.Add(1)
				dropNext.Store(p.dropped.CompareAndSwap(false, true))
			}
			if _, err := server.Write(buf[:n]); err != nil {
				return
			}
		}
	}()

	buf := make([]byte, 4096)
	for {
		n, err := server.Read(buf)
		if err != nil || dropNext.Load() {
			return
		}
		if _, err := client.Write(buf[:n]); err != nil {
			return
		}
	}
}

func TestCreateShortURLLostReply(t *testing.T) {
	for _, requestedKey := range []string{"custom", ""} {
		t.Run(fmt.Sprintf("key=%q", requestedKey), func(t *testing.T) {
			server := miniredis.RunT(t)
			proxy := newLostReplyProxy(t, server.Addr(), "set")
			mappings := newTestRedisMappingsAt(t, proxy.listener.Addr().String())
			generated := 0
			mappings.generateKey = func() (string, error) {
				generated++
				return fmt.Sprintf("generated%d", generated), nil
			}

			key, err := mappings.Create(context.Background(), "https://example.com", requestedKey)
			require.True(t, proxy.dropped.Load(), "the test must discard a SET reply")
			assert.Error(t, err, "a lost reply must remain a storage error")
			assert.NotErrorIs(t, err, errShortKeyExists, "our own write must not become a collision")
			assert.Empty(t, key)
			assert.EqualValues(t, 1, proxy.writes.Load(), "do not resend an uncertain write")
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

func TestResolveShortURLRetriesLostReply(t *testing.T) {
	server := miniredis.RunT(t)
	require.NoError(t, server.Set("custom", "https://example.com"))
	proxy := newLostReplyProxy(t, server.Addr(), "get")
	mappings := newTestRedisMappingsAt(t, proxy.listener.Addr().String())

	longURL, err := mappings.Resolve(context.Background(), "custom")
	require.True(t, proxy.dropped.Load(), "the test must discard a GET reply")
	require.NoError(t, err)
	require.Equal(t, "https://example.com", longURL)
	require.EqualValues(t, 2, proxy.writes.Load(), "a read is safe to resend")
}
