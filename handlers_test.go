package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLongToShortHandler(t *testing.T) {
	previousLogger := logger
	logger = zap.NewNop().Sugar()
	t.Cleanup(func() { logger = previousLogger })

	for _, name := range []string{"JSON", "legacy base64", "form", "generated key", "custom conflict", "retry exhaustion", "generation error", "storage error", "invalid parameters"} {
		t.Run(name, func(t *testing.T) {
			server, client := newTestRedis(t)
			creator := newTestCreator(t, client)
			creator.generateKey = func() (string, error) { return "autoKey", nil }
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
				require.NoError(t, client.Set(context.Background(), "custom", "original", time.Hour).Err())
				wantCode = ResponseCodeParamsCheckError
				wantMessage = "short key already exists, please use another one or leave it empty to generate automatically"
			case "retry exhaustion":
				params["shortKey"] = ""
				require.NoError(t, client.Set(context.Background(), "autoKey", "original", time.Hour).Err())
				wantCode = ResponseCodeServerError
				wantMessage = "failed to create short URL"
			case "storage error":
				server.SetError("ERR storage unavailable")
				wantCode = ResponseCodeServerError
				wantMessage = "failed to create short URL"
			case "generation error":
				params["shortKey"] = ""
				creator.generateKey = func() (string, error) {
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
			router.POST("/short", LongToShortHandler(creator))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			if wantCode == ResponseCodeSuccessLegacy {
				expected, err := json.Marshal(gin.H{"Code": 1, "ShortUrl": proto + "://" + domain + "/" + wantKey})
				require.NoError(t, err)
				require.JSONEq(t, string(expected), response.Body.String())
				require.Equal(t, longURL, client.Get(context.Background(), wantKey).Val())
			} else {
				expected, err := json.Marshal(Response{Code: wantCode, Msg: wantMessage})
				require.NoError(t, err)
				require.JSONEq(t, string(expected), response.Body.String())
			}
		})
	}
}
