package main

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed public/*
var publicFiles embed.FS

//go:embed public/index.html
var indexHTML []byte

func registerPublicRoutes(router *gin.Engine) {
	router.StaticFileFS("/logo.png", "public/logo.png", http.FS(publicFiles))
	router.StaticFileFS("/favicon.ico", "public/favicon.ico", http.FS(publicFiles))
	router.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}
