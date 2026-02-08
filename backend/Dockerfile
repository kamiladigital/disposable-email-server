# Multi-stage build
FROM golang:1.25.7-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application (disable cgo to avoid gcc dependency)
RUN CGO_ENABLED=0 GOOS=linux go build -o email-server ./cmd/email-server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates bash

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/email-server .

# Create emails directory
RUN mkdir -p emails

# Expose ports
EXPOSE 25 48080

# Set default HTTP_PORT
ENV HTTP_PORT=48080

# Run the server
CMD ["./email-server"]
