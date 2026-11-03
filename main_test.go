package main

import (
	"fmt"
	"net"
	"path/filepath"
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
	previousStorageType, previousSQLiteFile := storageType, sqliteFile
	t.Cleanup(func() {
		logger, redisAddr, port = previousLogger, previousAddr, previousPort
		storageType, sqliteFile = previousStorageType, previousSQLiteFile
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

func TestRunWithSQLiteNeedsNoRedis(t *testing.T) {
	prepareRun(t)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })

	redisAddr = "127.0.0.1:1"
	storageType = storageTypeSQLite
	sqliteFile = filepath.Join(t.TempDir(), "nested", "myurls.db")
	port = fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)

	// Reaching the listen error proves the storage came up.
	require.ErrorIs(t, run(), syscall.EADDRINUSE)
	require.FileExists(t, sqliteFile)
}

func TestRunRejectsUnsupportedStorageType(t *testing.T) {
	prepareRun(t)

	storageType = "memcached"

	require.ErrorContains(t, run(), `unsupported storage type "memcached"`)
}
