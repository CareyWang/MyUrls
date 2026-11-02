package main

import (
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRunReturnsListenError(t *testing.T) {
	previousLogger, previousClient, previousPort, previousMode := logger, RedisClient, port, gin.Mode()
	t.Cleanup(func() {
		logger, RedisClient, port = previousLogger, previousClient, previousPort
		gin.SetMode(previousMode)
	})
	t.Chdir(t.TempDir())

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })

	logger = zap.NewNop().Sugar()
	_, RedisClient = newTestRedis(t)
	port = fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)

	require.ErrorIs(t, run(), syscall.EADDRINUSE)
}
