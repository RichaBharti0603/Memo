package server

import (
	"fmt"
	"io"
	"strings"

	"redis_golang/internal/core"
	"redis_golang/internal/protocol/resp"
)

func readCommand(c io.ReadWriter) (*core.RedisCmd, error) {
	// Max read in one shot is 512 bytes
	var buf []byte = make([]byte, 512)
	n, err := c.Read(buf[:])
	if err != nil {
		return nil, err
	}

	tokens, err := resp.DecodeArrayString(buf[:n])
	if err != nil {
		return nil, err
	}

	return &core.RedisCmd{
		Cmd:  strings.ToUpper(tokens[0]),
		Args: tokens[1:],
	}, nil
}

func respondError(err error, c io.ReadWriter) {
	c.Write([]byte(fmt.Sprintf("-%s\r\n", err)))
}

func respond(cmd *core.RedisCmd, c io.ReadWriter) {
	err := core.EvalAndRespond(cmd, c)
	if err != nil {
		respondError(err, c)
	}
}

// FDComm implements io.ReadWriter for a file descriptor
type FDComm struct {
	Fd int
}

func (f FDComm) Write(b []byte) (int, error) {
	importSyscall := false
	_ = importSyscall // This will be handled in os-specific files if needed.
	// We'll leave FDComm here but implement its Write/Read in async_tcp_linux.go
	return 0, nil
}
func (f FDComm) Read(b []byte) (int, error) {
	return 0, nil
}
