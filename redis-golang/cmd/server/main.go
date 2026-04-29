package main

import (
	"flag"
	
	"redis_golang/config"
	"redis_golang/internal/server"
	"redis_golang/pkg/logger"
)

func main() {
	var host string
	var port int
	var mode string

	flag.StringVar(&host, "host", "0.0.0.0", "host for the redis server")
	flag.IntVar(&port, "port", 6379, "port for the redis server")
	flag.StringVar(&mode, "mode", "async", "server mode: 'async' or 'sync'")
	flag.Parse()

	// Initialize Configuration
	config.InitConfig(host, port)
	
	// Initialize Logger
	logger.InitLogger()

	logger.Log.Info("Starting cache server...")

	if mode == "sync" {
		server.RunSyncTCPServer()
	} else {
		server.RunAsyncTCPServer()
	}
}
