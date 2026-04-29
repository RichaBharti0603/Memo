//go:build !linux

package server

import "redis_golang/pkg/logger"

func RunAsyncTCPServer() {
	logger.Log.Warn("Async TCP server via epoll is only supported on Linux. Falling back to Sync TCP Server.")
	RunSyncTCPServer()
}
