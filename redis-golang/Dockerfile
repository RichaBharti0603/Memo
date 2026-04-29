# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Go application with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/server ./cmd/server

# Stage 2: Create a minimal production image
FROM alpine:latest

WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/bin/server .

# Expose ports
EXPOSE 6379
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["./server"]
CMD ["--mode=async", "--web-port=8080"]