# Dockerfile for running the Go application on AKS
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN go build -o cosmos-app main.go

# Use a minimal base image for runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/cosmos-app .

# Expose metrics port
EXPOSE 8080

# Run the application
CMD ["./cosmos-app"]
