# Build stage
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod ./
# Copy go.sum if it exists, otherwise create an empty one
COPY go.su[m] ./

# Copy source code first to get all dependencies
COPY . .

# Ensure dependencies are up to date and download them
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy migrations folder (needed for automatic migrations)
COPY --from=builder /app/database/migrations ./database/migrations

# Change ownership to non-root user
RUN chown -R appuser:appgroup /root/

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]
