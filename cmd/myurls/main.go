package main

import (
	"log"

	"github.com/CareyWang/MyUrls/internal/bootstrap"
)

func main() {
	app := bootstrap.New()
	if err := app.Run(); err != nil {
		log.Fatalf("app run failed: %v", err)
	}
}
