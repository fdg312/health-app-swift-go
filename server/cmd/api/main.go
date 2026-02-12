package main

import (
	"log"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/httpserver"
)

func main() {
	cfg := config.Load()

	server := httpserver.New(cfg)

	log.Fatal(server.Start())
}
