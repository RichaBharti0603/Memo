package main

import (
	"flag"
	"fmt"
	"os"

	"redis_golang/config"
	"redis_golang/internal/server"
	"redis_golang/pkg/logger"
	"redis_golang/ui/tui"
)

func main() {
	var host string
	var port int
	var mode string
	var useTUI bool

	flag.StringVar(&host, "host", "0.0.0.0", "host for the redis server")
	flag.IntVar(&port, "port", 6379, "port for the redis server")
	flag.StringVar(&mode, "mode", "async", "server mode: 'async' or 'sync'")
	flag.BoolVar(&useTUI, "tui", false, "run the terminal UI dashboard")
	flag.Parse()

	// Initialize Configuration
	config.InitConfig(host, port)
	
	// Initialize Logger
	logger.InitLogger(useTUI)

	logger.Log.Info("Starting cache server...")

	// Run server in a goroutine if TUI is enabled, else run blocking
	if useTUI {
		go func() {
			if mode == "sync" {
				server.RunSyncTCPServer()
			} else {
				server.RunAsyncTCPServer()
			}
		}()
		
		// Run TUI blocking
		if err := tui.StartApp(); err != nil {
			fmt.Printf("Error starting TUI: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Run Server blocking
		if mode == "sync" {
			server.RunSyncTCPServer()
		} else {
			server.RunAsyncTCPServer()
		}
	}
}
