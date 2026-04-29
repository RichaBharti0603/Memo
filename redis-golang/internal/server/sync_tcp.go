package server

import (
	"io"
	"net"
	"strconv"

	"redis_golang/config"
	"redis_golang/internal/storage/memory"
	"redis_golang/pkg/logger"
)

func RunSyncTCPServer() {
	logger.Log.Info("starting a synchronous TCP server", "host", config.GlobalConfig.Host, "port", config.GlobalConfig.Port)

	memory.StartCleanupRoutine()

	var conClients int = 0

	addr := config.GlobalConfig.Host + ":" + strconv.Itoa(config.GlobalConfig.Port)
	lsnr, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Log.Error("failed to listen", "error", err)
		return
	}

	for {
		c, err := lsnr.Accept()
		if err != nil {
			logger.Log.Error("failed to accept connection", "error", err)
			continue
		}

		conClients++
		logger.Log.Info("client connected", "total_clients", conClients)

		go func(conn net.Conn) {
			defer func() {
				conn.Close()
				conClients--
				logger.Log.Info("client disconnected", "total_clients", conClients)
			}()

			for {
				cmd, err := readCommand(conn)
				if err != nil {
					if err != io.EOF {
						logger.Log.Error("read error", "error", err)
					}
					break
				}
				respond(cmd, conn)
			}
		}(c)
	}
}
