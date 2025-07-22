# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pr-documentator cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS calls and openssl for certificate generation
RUN apk --no-cache add ca-certificates openssl

# Create app user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/pr-documentator .

# Copy scripts
COPY scripts/ ./scripts/
RUN chmod +x ./scripts/generate_certs.sh

# Create directories
RUN mkdir -p certs logs

# Change ownership to app user
RUN chown -R appuser:appuser /app

# Switch to app user
USER appuser

# Generate certificates on startup (development only)
RUN ./scripts/generate_certs.sh

# Expose port
EXPOSE 8443

# Run the application
CMD ["./pr-documentator"]