package replication

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"redis_golang/internal/metrics"
	"redis_golang/internal/storage/memory"
	"redis_golang/pkg/logger"
)

var (
	replicaConns []io.ReadWriter
	replicaMu    sync.RWMutex
)

func AddReplica(c io.ReadWriter) {
	replicaMu.Lock()
	defer replicaMu.Unlock()
	replicaConns = append(replicaConns, c)
	metrics.IncReplica()
}

func Broadcast(cmd string, args []string) {
	if metrics.GetConnectedReplicas() == 0 {
		return
	}

	tokens := append([]string{cmd}, args...)
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("*%d\r\n", len(tokens)))
	for _, t := range tokens {
		buf.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(t), t))
	}
	data := buf.Bytes()

	replicaMu.Lock()
	defer replicaMu.Unlock()

	var active []io.ReadWriter
	for _, c := range replicaConns {
		_, err := c.Write(data)
		if err != nil {
			logger.Log.Error("Failed to broadcast to replica, removing connection", "error", err)
			metrics.DecReplica()
			// don't add to active
		} else {
			active = append(active, c)
		}
	}
	replicaConns = active
}

func HandleSync(c io.ReadWriter) error {
	logger.Log.Info("Starting SYNC for new replica")
	
	// 1. Add to replica pool immediately so it gets all new commands
	AddReplica(c)

	// 2. Iterate memory and send SET/HSET commands
	keys := memory.GetAllKeys()
	for _, k := range keys {
		obj := memory.Get(k)
		if obj == nil {
			continue
		}
		
		// If it's a string
		if strVal, ok := obj.Value.(string); ok {
			tokens := []string{"SET", k, strVal}
			var buf bytes.Buffer
			buf.WriteString(fmt.Sprintf("*%d\r\n", len(tokens)))
			for _, t := range tokens {
				buf.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(t), t))
			}
			c.Write(buf.Bytes())
		}
		// If it's a map
		if hashVal, ok := obj.Value.(map[string]string); ok {
			for field, val := range hashVal {
				tokens := []string{"HSET", k, field, val}
				var buf bytes.Buffer
				buf.WriteString(fmt.Sprintf("*%d\r\n", len(tokens)))
				for _, t := range tokens {
					buf.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(t), t))
				}
				c.Write(buf.Bytes())
			}
		}
	}

	logger.Log.Info("SYNC full state transfer complete")
	return nil
}
