# Redis-Golang Cache Server

A high-performance, distributed, and persistence-ready key-value store built in Go, engineered to handle massive throughput with enterprise-grade interfaces.

![Redis-Golang Dashboard](ui/web/static/style.css) 

## 🚀 Features

- **Blazing Fast Backend:** Custom thread-safe `map[string]interface{}` with `sync.RWMutex`.
- **Async I/O:** Uses Linux `epoll` for non-blocking I/O multiplexing capable of handling 20,000+ concurrent connections efficiently (with a cross-platform Sync fallback).
- **AOF Persistence:** Guaranteed durability with Append-Only File support. It automatically replays historical commands on startup to recover state.
- **Embedded Web Dashboard:** A beautiful, dark-mode real-time monitoring interface embedded right into the Go binary. No extra dependencies required!
- **Terminal UI (TUI):** A hacker-style `bubbletea` dashboard allowing you to monitor metrics right from your shell.
- **Advanced Commands:** Supports `GET`, `SET`, `DEL`, `TTL`, `PING`, along with numerics (`INCR`, `DECR`) and Hashes (`HSET`, `HGET`).
- **Clean Architecture:** Fully modularized into `internal/core`, `storage`, `server`, and `protocol` layers.

## 📦 Quick Start

### 1. Run using Docker (Recommended)
We provide a minimal multi-stage Docker build:
```bash
docker-compose up -d --build
```
This maps TCP port `6379` for your Redis clients and port `8080` for the Web Dashboard.

### 2. Run Locally from Source
Make sure you have Go 1.24+ installed.
```bash
go build -o bin/server ./cmd/server
./bin/server --mode=async --tui=true --web-port=8080
```
- `--mode=async` activates the Epoll server (Linux only). Use `--mode=sync` on Windows/Mac.
- `--tui=true` starts the integrated Bubble Tea dashboard.
- `--web-port=8080` starts the embedded HTTP Web UI.

## 🖥️ Dashboards

### Web Dashboard
Once the server is running, navigate to [http://localhost:8080](http://localhost:8080) to view the live Grafana-style dashboard.

### TUI Dashboard
Running with `--tui=true` renders a live terminal interface displaying active connections, cache hit rates, memory limits, and real-time logs.

## 🛠️ Architecture

```
redis-golang/
├── cmd/server/       # Entry point
├── internal/
│   ├── core/         # Command definitions and evaluation router
│   ├── metrics/      # Atomic counters for active stats
│   ├── protocol/     # RESP (Redis Serialization Protocol) parser
│   ├── server/       # TCP servers (Async Epoll + Sync fallback)
│   └── storage/      # Memory store and AOF persistence engine
├── pkg/logger/       # Custom slog wrapper and Memory Ring Buffer handler
└── ui/
    ├── tui/          # Bubble Tea terminal interface
    └── web/          # Embedded static HTML/CSS/JS frontend
```

## 📖 Command Reference

| Command | Usage | Description |
|---|---|---|
| `SET` | `SET key value [EX seconds]` | Set a key with an optional TTL. |
| `GET` | `GET key` | Retrieve a key. |
| `DEL` | `DEL key [key ...]` | Delete one or more keys. |
| `TTL` | `TTL key` | Get remaining TTL in seconds. |
| `INCR` | `INCR key` | Increment the integer value of a key. |
| `DECR` | `DECR key` | Decrement the integer value of a key. |
| `HSET` | `HSET key field value` | Set the string value of a hash field. |
| `HGET` | `HGET key field` | Get the value of a hash field. |
| `PING` | `PING [message]` | Ping the server. |
