# Stage 1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install certificates for HTTPS scanning and cURL for health checks
RUN apk --no-cache add ca-certificates curl

# Copy binary from builder
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
