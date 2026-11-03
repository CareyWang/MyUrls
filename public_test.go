package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPublicRoutesWithoutWorkingDirectoryAssets(t *testing.T) {
	index, err := os.ReadFile("public/index.html")
	require.NoError(t, err)
	logo, err := os.ReadFile("public/logo.png")
	require.NoError(t, err)
	favicon, err := os.ReadFile("public/favicon.ico")
	require.NoError(t, err)

	t.Chdir(t.TempDir())
	router := gin.New()
	registerPublicRoutes(router)
	router.GET("/:shortKey", func(c *gin.Context) {
		c.String(http.StatusOK, c.Param("shortKey"))
	})

	// The content type of .ico comes from the system's MIME table, so only the body is compared.
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/favicon.ico", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, favicon, response.Body.Bytes())

	for _, tc := range []struct {
		name        string
		method      string
		path        string
		contentType string
		body        []byte
	}{
		{"homepage", http.MethodGet, "/", "text/html; charset=utf-8", index},
		{"logo", http.MethodGet, "/logo.png", "image/png", logo},
		{"logo head", http.MethodHead, "/logo.png", "image/png", nil},
		{"short key route", http.MethodGet, "/example", "text/plain; charset=utf-8", []byte("example")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			require.Equal(t, http.StatusOK, response.Code)
			require.Equal(t, tc.contentType, response.Header().Get("Content-Type"))
			require.Equal(t, tc.body, response.Body.Bytes())
		})
	}
}
