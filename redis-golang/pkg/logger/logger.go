package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger
var MemHandler *MemoryHandler

func InitLogger(useTUI bool) {
	opts := slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	
	var baseHandler slog.Handler
	if !useTUI {
		baseHandler = slog.NewJSONHandler(os.Stdout, &opts)
	}
	
	MemHandler = NewMemoryHandler(100, baseHandler, opts)
	Log = slog.New(MemHandler)
	slog.SetDefault(Log)
}
