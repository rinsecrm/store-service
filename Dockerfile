# Build stage
FROM golang:1.21-alpine AS builder

# Install git, ca-certificates, and protoc tools
RUN apk add --no-cache git ca-certificates protobuf-dev
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Copy proto directory first (needed for go mod download due to replace directive)
COPY proto/ ./proto/

# Download dependencies
RUN go mod download

# Generate protobuf code
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/store.proto

# Copy source code
COPY . .

# Build the server with static linking and security optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o store-service ./cmd/server

# Extract CA certificates for the final image
RUN mkdir -p /ca-certs && \
    cp /etc/ssl/certs/ca-certificates.crt /ca-certs/

# Final stage - start from scratch for maximum security
FROM scratch

# Copy CA certificates from builder stage
COPY --from=builder /ca-certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the binary from builder stage
COPY --from=builder /app/store-service /store-service

# Set working directory
WORKDIR /

# Expose port
EXPOSE 8080

# Run the server
CMD ["/store-service"]
