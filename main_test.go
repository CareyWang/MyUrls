package main

import (
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func prepareRun(t *testing.T) {
	t.Helper()
	previousLogger, previousAddr, previousPort, previousMode := logger, redisAddr, port, gin.Mode()
	t.Cleanup(func() {
		logger, redisAddr, port = previousLogger, previousAddr, previousPort
		gin.SetMode(previousMode)
	})
	t.Chdir(t.TempDir())
	logger = zap.NewNop().Sugar()
}

func TestRunReturnsListenError(t *testing.T) {
	prepareRun(t)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })

	redisAddr = miniredis.RunT(t).Addr()
	port = fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)

	require.ErrorIs(t, run(), syscall.EADDRINUSE)
}

func TestRunReturnsRedisPingError(t *testing.T) {
	prepareRun(t)

	server := miniredis.RunT(t)
	redisAddr = server.Addr()
	server.Close()

	require.ErrorContains(t, run(), "redis ping failed")
}
