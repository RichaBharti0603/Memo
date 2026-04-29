package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"redis_golang/config"
	"redis_golang/internal/core"
	"redis_golang/internal/server"
	"redis_golang/internal/replication"
	"redis_golang/internal/storage/persistence"
	"redis_golang/pkg/logger"
	"redis_golang/ui/tui"
)

func main() {
	var host string
	var port int
	var mode string
	var useTUI bool
	var webPort int
	var replicaHost string
	var replicaPort int

	flag.StringVar(&host, "host", "0.0.0.0", "host for the redis server")
	flag.IntVar(&port, "port", 6379, "port for the redis server")
	flag.StringVar(&mode, "mode", "async", "server mode: 'async' or 'sync'")
	flag.BoolVar(&useTUI, "tui", false, "run the terminal UI dashboard")
	flag.IntVar(&webPort, "web-port", 8080, "port for the web dashboard (0 to disable)")
	flag.StringVar(&replicaHost, "replica-host", "", "host of the master to replicate from (if any)")
	flag.IntVar(&replicaPort, "replica-port", 0, "port of the master to replicate from (if any)")
	flag.Parse()

	// Initialize Configuration
	config.InitConfig(host, port, replicaHost, replicaPort)
	
	// Initialize Logger
	logger.InitLogger(useTUI)

	// Initialize AOF Persistence
	if err := persistence.InitAOF("appendonly.aof"); err != nil {
		logger.Log.Error("Failed to initialize AOF", "error", err)
	} else {
		persistence.ReplayAOF(func(cmd string, args []string, c io.ReadWriter) error {
			return core.EvalAndRespond(&core.RedisCmd{Cmd: cmd, Args: args}, c)
		})
	}

	// Start replication background thread if replica mode is requested via flag
	if replicaHost != "" && replicaPort > 0 {
		go replication.StartReplica(replicaHost, replicaPort, func(cmd string, args []string, c io.ReadWriter) error {
			return core.EvalCommandUnsafe(&core.RedisCmd{Cmd: cmd, Args: args}, c)
		})
	}

	logger.Log.Info("Starting cache server...")

	// Start Web Dashboard if enabled
	if webPort > 0 {
		go server.RunHTTPServer(webPort)
	}

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
