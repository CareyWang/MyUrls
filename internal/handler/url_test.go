package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CareyWang/MyUrls/internal/config"
	"github.com/CareyWang/MyUrls/internal/logger"
	"github.com/CareyWang/MyUrls/internal/model"
	"github.com/CareyWang/MyUrls/internal/storage"
)

func setupTestEnvironment(t *testing.T) (*URLHandler, *config.Config) {
	// 初始化日志
	logger.Init()

	// 初始化存储 - 使用SQLite内存数据库用于测试
	storageConfig := &config.StorageConfig{
		Type:       config.StorageSQLite,
		SQLiteFile: ":memory:", // 使用内存数据库
	}

	err := storage.InitStorage(storageConfig)
	require.NoError(t, err)

	// 创建测试配置
	cfg := &config.Config{
		Server: config.ServerConfig{
			Proto:  "https",
			Domain: "test.example.com",
		},
	}

	// 创建URLHandler实例
	handler := NewURLHandler(cfg)

	return handler, cfg
}

func TestShortToLongHandler(t *testing.T) {
	handler, _ := setupTestEnvironment(t)

	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		shortKey         string
		setupData        func(t *testing.T) // 预设数据
		expectedStatus   int
		expectedRedirect string
		expectError      bool
	}{
		{
			name:     "successful redirect",
			shortKey: "test123",
			setupData: func(t *testing.T) {
				// 预先插入测试数据
				driver := storage.GetDriver()
				ctx := context.Background()
				err := driver.SetEx(ctx, "test123", "https://www.google.com", 3600*time.Second)
				require.NoError(t, err)
			},
			expectedStatus:   http.StatusMovedPermanently,
			expectedRedirect: "https://www.google.com",
			expectError:      false,
		},
		{
			name:           "short key not found",
			shortKey:       "notexist",
			setupData:      func(t *testing.T) {}, // 不插入数据
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试数据
			tt.setupData(t)

			// 创建测试路由
			router := gin.New()
			router.GET("/:shortKey", handler.ShortToLongHandler())

			// 创建测试请求
			req, err := http.NewRequest("GET", "/"+tt.shortKey, nil)
			require.NoError(t, err)

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			router.ServeHTTP(w, req)

			// 检查结果
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectError {
				// 检查错误响应
				var resp model.Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, model.ResponseCodeServerError, resp.Code)
			} else {
				// 检查重定向
				assert.Equal(t, tt.expectedRedirect, w.Header().Get("Location"))
			}
		})
	}
}

func TestLongToShortHandler(t *testing.T) {
	handler, cfg := setupTestEnvironment(t)

	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		contentType    string
		expectedStatus int
		expectSuccess  bool
		expectedError  string
	}{
		{
			name: "successful creation with JSON",
			requestBody: LongToShortParams{
				LongUrl:  "https://www.example.com",
				ShortKey: "",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "successful creation with custom short key",
			requestBody: LongToShortParams{
				LongUrl:  "https://www.example.com/custom",
				ShortKey: "custom1",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "successful creation with form data",
			requestBody:    "longUrl=https://www.example.com/form&shortKey=",
			contentType:    "application/x-www-form-urlencoded",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "missing required parameter",
			requestBody: LongToShortParams{
				ShortKey: "test",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectSuccess:  false,
			expectedError:  "invalid parameters",
		},
		{
			name: "duplicate short key",
			requestBody: LongToShortParams{
				LongUrl:  "https://www.example.com/duplicate",
				ShortKey: "duplicate",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectSuccess:  false,
			expectedError:  "short key already exists",
		},
	}

	// 预先插入一个重复的key用于测试
	driver := storage.GetDriver()
	ctx := context.Background()
	err := driver.SetEx(ctx, "duplicate", "https://existing.com", 3600*time.Second)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试路由
			router := gin.New()
			router.POST("/api/short", handler.LongToShortHandler())

			var req *http.Request
			var err error

			// 根据内容类型创建请求
			if tt.contentType == "application/json" {
				jsonBody, err := json.Marshal(tt.requestBody)
				require.NoError(t, err)
				req, err = http.NewRequest("POST", "/api/short", bytes.NewBuffer(jsonBody))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")
			} else if tt.contentType == "application/x-www-form-urlencoded" {
				req, err = http.NewRequest("POST", "/api/short", strings.NewReader(tt.requestBody.(string)))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 执行请求
			router.ServeHTTP(w, req)

			// 检查HTTP状态码
			assert.Equal(t, tt.expectedStatus, w.Code)

			// 解析响应
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectSuccess {
				// 检查成功响应
				assert.Equal(t, float64(model.ResponseCodeSuccessLegacy), response["Code"])
				shortUrl, exists := response["ShortUrl"]
				assert.True(t, exists)
				assert.Contains(t, shortUrl.(string), cfg.Server.Proto+"://"+cfg.Server.Domain+"/")
			} else {
				// 检查错误响应
				if response["Code"] != nil {
					code := response["Code"].(float64)
					assert.NotEqual(t, float64(model.ResponseCodeSuccessLegacy), code)
				}
				if tt.expectedError != "" {
					msg, exists := response["Msg"]
					if exists {
						assert.Contains(t, msg.(string), tt.expectedError)
					}
				}
			}
		})
	}
}

func TestLongToShortHandlerWithBase64(t *testing.T) {
	handler, cfg := setupTestEnvironment(t)

	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 创建测试路由
	router := gin.New()
	router.POST("/api/short", handler.LongToShortHandler())

	// 测试Base64编码的URL
	originalUrl := "https://www.example.com/test"
	base64Url := "aHR0cHM6Ly93d3cuZXhhbXBsZS5jb20vdGVzdA==" // base64 encoded

	requestBody := LongToShortParams{
		LongUrl:  base64Url,
		ShortKey: "",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/short", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 检查响应
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 应该成功创建短链接
	assert.Equal(t, float64(model.ResponseCodeSuccessLegacy), response["Code"])

	// 验证存储的是解码后的URL
	shortUrl := response["ShortUrl"].(string)
	shortKey := strings.TrimPrefix(shortUrl, cfg.Server.Proto+"://"+cfg.Server.Domain+"/")

	driver := storage.GetDriver()
	ctx := context.Background()
	storedUrl, err := driver.Get(ctx, shortKey)
	require.NoError(t, err)
	assert.Equal(t, originalUrl, storedUrl)
}

func TestHandlerConstants(t *testing.T) {
	// 测试默认常量
	assert.Equal(t, time.Hour*24*365, defaultTTL)
	assert.Equal(t, time.Hour*48, defaultRenewTime)
	assert.Equal(t, 7, defaultShortKeyLength)
}
