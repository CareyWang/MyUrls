package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLongToShortHandlerRejectsInvalidInput(t *testing.T) {
	previousLogger := logger
	logger = zap.NewNop().Sugar()
	t.Cleanup(func() { logger = previousLogger })

	const invalidLongURL = "invalid long URL, only http and https URLs are supported"
	const invalidShortKey = "invalid short key, only letters, digits, hyphens, and underscores are allowed"
	for _, tc := range []struct {
		name        string
		longURL     string
		shortKey    string
		wantMessage string
	}{
		{"plain text decodable as base64", "example/path", "", invalidLongURL},
		{"missing scheme", "example.com", "", invalidLongURL},
		{"javascript scheme", "javascript:alert(1)", "", invalidLongURL},
		{"base64 javascript scheme", base64.StdEncoding.EncodeToString([]byte("javascript:alert(1)")), "", invalidLongURL},
		{"missing host", "https://", "", invalidLongURL},
		{"short key shadows static file", "https://example.com", "logo.png", invalidShortKey},
		{"short key with slash", "https://example.com", "a/b", invalidShortKey},
		{"short key with query", "https://example.com", "q?x", invalidShortKey},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, mappings := newTestMappings(t)
			body, err := json.Marshal(map[string]string{"longUrl": tc.longURL, "shortKey": tc.shortKey})
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/short", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			router := gin.New()
			router.POST("/short", LongToShortHandler(mappings))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			expected, err := json.Marshal(Response{Code: ResponseCodeParamsCheckError, Msg: tc.wantMessage})
			require.NoError(t, err)
			require.JSONEq(t, string(expected), response.Body.String())
			require.Empty(t, server.Keys())
		})
	}
}

func TestShortToLongHandler(t *testing.T) {
	previousLogger := logger
	logger = zap.NewNop().Sugar()
	t.Cleanup(func() { logger = previousLogger })

	server, mappings := newTestMappings(t)
	require.NoError(t, server.Set("custom", "https://example.com"))
	router := gin.New()
	router.GET("/:shortKey", ShortToLongHandler(mappings))
	get := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}

	response := get("/custom")
	require.Equal(t, http.StatusMovedPermanently, response.Code)
	require.Equal(t, "https://example.com", response.Header().Get("Location"))

	response = get("/missing")
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Contains(t, response.Body.String(), "exists or expired")

	server.SetError("ERR storage unavailable")
	response = get("/custom")
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.NotContains(t, response.Body.String(), "exists or expired")
	require.NotContains(t, response.Body.String(), "storage unavailable")
}

func TestLongToShortHandler(t *testing.T) {
	previousLogger := logger
	logger = zap.NewNop().Sugar()
	t.Cleanup(func() { logger = previousLogger })

	for _, name := range []string{"JSON", "legacy base64", "form", "generated key", "custom conflict", "retry exhaustion", "generation error", "storage error", "invalid parameters"} {
		t.Run(name, func(t *testing.T) {
			server, mappings := newTestMappings(t)
			mappings.generateKey = func() (string, error) { return "autoKey", nil }
			longURL := "https://example.com/path?a=1"
			params := map[string]string{"longUrl": longURL, "shortKey": "custom"}
			wantKey := "custom"
			wantCode := ResponseCodeSuccessLegacy
			wantMessage := ""
			switch name {
			case "legacy base64":
				params["longUrl"] = base64.StdEncoding.EncodeToString([]byte(longURL))
			case "generated key":
				params["shortKey"] = ""
				wantKey = "autoKey"
			case "custom conflict":
				require.NoError(t, server.Set("custom", "original"))
				wantCode = ResponseCodeParamsCheckError
				wantMessage = "short key already exists, please use another one or leave it empty to generate automatically"
			case "retry exhaustion":
				params["shortKey"] = ""
				require.NoError(t, server.Set("autoKey", "original"))
				wantCode = ResponseCodeServerError
				wantMessage = "failed to create short URL"
			case "storage error":
				server.SetError("ERR storage unavailable")
				wantCode = ResponseCodeServerError
				wantMessage = "failed to create short URL"
			case "generation error":
				params["shortKey"] = ""
				mappings.generateKey = func() (string, error) {
					return "", errors.New("random source unavailable")
				}
				wantCode = ResponseCodeServerError
				wantMessage = "failed to create short URL"
			case "invalid parameters":
				delete(params, "longUrl")
				wantCode = ResponseCodeParamsCheckError
				wantMessage = "invalid parameters"
			}

			body, err := json.Marshal(params)
			require.NoError(t, err)
			contentType := "application/json"
			if name == "form" {
				body = []byte(url.Values{"longUrl": {longURL}, "shortKey": {"custom"}}.Encode())
				contentType = "application/x-www-form-urlencoded"
			}
			request := httptest.NewRequest(http.MethodPost, "/short", bytes.NewReader(body))
			request.Header.Set("Content-Type", contentType)
			router := gin.New()
			router.POST("/short", LongToShortHandler(mappings))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			if wantCode == ResponseCodeSuccessLegacy {
				expected, err := json.Marshal(gin.H{"Code": 1, "ShortUrl": proto + "://" + domain + "/" + wantKey})
				require.NoError(t, err)
				require.JSONEq(t, string(expected), response.Body.String())
				requireMapping(t, mappings, wantKey, longURL)
			} else {
				expected, err := json.Marshal(Response{Code: wantCode, Msg: wantMessage})
				require.NoError(t, err)
				require.JSONEq(t, string(expected), response.Body.String())
			}
		})
	}
}
