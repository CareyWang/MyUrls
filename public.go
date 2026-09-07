package main

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed public/*
var publicFiles embed.FS

func registerPublicRoutes(router *gin.Engine) {
	router.SetHTMLTemplate(template.Must(template.ParseFS(publicFiles, "public/*.html")))
	router.StaticFileFS("/logo.png", "public/logo.png", http.FS(publicFiles))
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{"title": "MyUrls"})
	})
}
