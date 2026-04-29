package replication

import (
	"fmt"
	"net"
	"time"
	"io"
	"strings"
	"redis_golang/internal/protocol/resp"
	"redis_golang/pkg/logger"
)

var GlobalRole Role = RolePrimary

func StartReplica(masterHost string, masterPort int, evalFunc func(cmd string, args []string, c io.ReadWriter) error) {
	GlobalRole = RoleReplica
	
	addr := fmt.Sprintf("%s:%d", masterHost, masterPort)
	logger.Log.Info("Starting replica mode, connecting to master", "master", addr)

	for {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			logger.Log.Error("Failed to connect to master, retrying in 5s", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		logger.Log.Info("Connected to master. Requesting SYNC...")
		// Send SYNC
		conn.Write([]byte("*1\r\n$4\r\nSYNC\r\n"))

		// Read loop
		buf := make([]byte, 0)
		tmp := make([]byte, 1024)
		
		for {
			n, err := conn.Read(tmp)
			if err != nil {
				if err != io.EOF {
					logger.Log.Error("Master connection error", "error", err)
				}
				break
			}
			buf = append(buf, tmp[:n]...)

			for len(buf) > 0 {
				val, delta, err := resp.DecodeOne(buf)
				if err != nil {
					// Incomplete or error
					break
				}
				buf = buf[delta:]

				if arr, ok := val.([]interface{}); ok {
					tokens := make([]string, len(arr))
					for i, v := range arr {
						tokens[i] = v.(string)
					}
					if len(tokens) > 0 {
						// We pass a dummy writer since replicas don't respond to master
						evalFunc(strings.ToUpper(tokens[0]), tokens[1:], &dummyWriter{})
					}
				}
			}
		}
		logger.Log.Error("Disconnected from master. Reconnecting...")
		time.Sleep(2 * time.Second)
	}
}

type dummyWriter struct{}

func (d *dummyWriter) Read(p []byte) (n int, err error) { return 0, io.EOF }
func (d *dummyWriter) Write(p []byte) (n int, err error) { return len(p), nil }
