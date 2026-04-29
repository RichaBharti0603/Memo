//go:build linux

package server

import (
	"net"
	"syscall"

	"redis_golang/config"
	"redis_golang/internal/metrics"
	"redis_golang/internal/storage/memory"
	"redis_golang/pkg/logger"
)

type FDComm struct {
	Fd int
}

func (f FDComm) Write(b []byte) (int, error) {
	return syscall.Write(f.Fd, b)
}

func (f FDComm) Read(b []byte) (int, error) {
	return syscall.Read(f.Fd, b)
}

func RunAsyncTCPServer() {
	logger.Log.Info("starting an asynchronous TCP server", "host", config.GlobalConfig.Host, "port", config.GlobalConfig.Port)

	memory.StartCleanupRoutine()

	maxClients := 20000
	events := make([]syscall.EpollEvent, maxClients)

	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.O_NONBLOCK|syscall.SOCK_STREAM, 0)
	if err != nil {
		logger.Log.Error("failed to create socket", "error", err)
		return
	}
	defer syscall.Close(serverFD)

	if err = syscall.SetNonblock(serverFD, true); err != nil {
		logger.Log.Error("failed to set nonblock", "error", err)
		return
	}

	ip4 := net.ParseIP(config.GlobalConfig.Host)
	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: config.GlobalConfig.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		logger.Log.Error("failed to bind", "error", err)
		return
	}

	if err = syscall.Listen(serverFD, maxClients); err != nil {
		logger.Log.Error("failed to listen", "error", err)
		return
	}

	epollFD, err := syscall.EpollCreate1(0)
	if err != nil {
		logger.Log.Error("failed to create epoll", "error", err)
		return
	}
	defer syscall.Close(epollFD)

	socketServerEvent := syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(serverFD),
	}

	if err = syscall.EpollCtl(epollFD, syscall.EPOLL_CTL_ADD, serverFD, &socketServerEvent); err != nil {
		logger.Log.Error("failed to add server event to epoll", "error", err)
		return
	}

	conClients := 0

	for {
		nevents, e := syscall.EpollWait(epollFD, events[:], -1)
		if e != nil {
			continue
		}

		for i := 0; i < nevents; i++ {
			if int(events[i].Fd) == serverFD {
				fd, _, err := syscall.Accept(serverFD)
				if err != nil {
					logger.Log.Error("failed to accept", "error", err)
					continue
				}

				conClients++
				metrics.IncConn()
				logger.Log.Info("new client connected", "total_clients", metrics.GetActiveConnections())
				syscall.SetNonblock(fd, true)

				socketClientEvent := syscall.EpollEvent{
					Events: syscall.EPOLLIN | syscall.EPOLLRDHUP,
					Fd:     int32(fd),
				}
				if err := syscall.EpollCtl(epollFD, syscall.EPOLL_CTL_ADD, fd, &socketClientEvent); err != nil {
					logger.Log.Error("failed to add client event to epoll", "error", err)
				}
			} else {
				if events[i].Events&syscall.EPOLLRDHUP != 0 {
					logger.Log.Info("client disconnected", "fd", events[i].Fd)
					syscall.Close(int(events[i].Fd))
					conClients--
					metrics.DecConn()
					continue
				}

				comm := FDComm{Fd: int(events[i].Fd)}
				cmd, err := readCommand(comm)
				if err != nil {
					logger.Log.Error("client read error", "error", err)
					syscall.Close(int(events[i].Fd))
					conClients--
					metrics.DecConn()
					continue
				}
				
				logger.Log.Debug("received command", "cmd", cmd.Cmd, "args", cmd.Args)
				respond(cmd, comm)
			}
		}
	}
}
