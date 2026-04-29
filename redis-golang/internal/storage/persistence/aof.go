package persistence

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"redis_golang/internal/protocol/resp"
	"redis_golang/pkg/logger"
)

type AOF struct {
	file *os.File
	mu   sync.Mutex
}

var GlobalAOF *AOF
var IsReplaying bool = false

func InitAOF(filename string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	GlobalAOF = &AOF{file: f}
	return nil
}

func (a *AOF) WriteCmd(cmd string, args []string) error {
	if IsReplaying {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Reconstruct the RESP array representation of the command
	// e.g. SET key value -> *3\r\n$3\r\nSET\r\n...
	
	tokens := append([]string{cmd}, args...)
	
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("*%d\r\n", len(tokens)))
	for _, t := range tokens {
		buf.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(t), t))
	}

	_, err := a.file.Write(buf.Bytes())
	return err
}

func (a *AOF) Close() error {
	if a.file != nil {
		return a.file.Close()
	}
	return nil
}

// ReplayAOF reads the AOF file and re-executes all commands
func ReplayAOF(evalFunc func(cmd string, args []string, c io.ReadWriter) error) {
	if GlobalAOF == nil {
		return
	}
	
	GlobalAOF.mu.Lock()
	defer GlobalAOF.mu.Unlock()

	GlobalAOF.file.Seek(0, 0)
	
	// We read using resp decoder
	// Since we don't have a stream reader in our resp package, we load the whole file for now
	// For production, we should implement a stream reader
	data, err := io.ReadAll(GlobalAOF.file)
	if err != nil || len(data) == 0 {
		return
	}

	logger.Log.Info("Replaying AOF file to rebuild state...")
	IsReplaying = true
	defer func() { IsReplaying = false }()
	
	pos := 0
	count := 0
	
	// A mock connection that discards output during replay
	mockConn := &discardReadWriter{}

	for pos < len(data) {
		if data[pos] != '*' {
			break
		}
		
		// This uses our existing DecodeOne function which returns the parsed element and how many bytes it consumed
		val, delta, err := resp.DecodeOne(data[pos:])
		if err != nil {
			logger.Log.Error("Error replaying AOF", "error", err)
			break
		}
		
		pos += delta

		if arr, ok := val.([]interface{}); ok {
			tokens := make([]string, len(arr))
			for i, v := range arr {
				tokens[i] = v.(string)
			}
			if len(tokens) > 0 {
				evalFunc(strings.ToUpper(tokens[0]), tokens[1:], mockConn)
				count++
			}
		}
	}
	logger.Log.Info("AOF replay complete", "commands_replayed", count)
	
	// Reset file pointer to end for future appends
	GlobalAOF.file.Seek(0, io.SeekEnd)
}

type discardReadWriter struct{}

func (d *discardReadWriter) Read(p []byte) (n int, err error) { return 0, io.EOF }
func (d *discardReadWriter) Write(p []byte) (n int, err error) { return len(p), nil }
