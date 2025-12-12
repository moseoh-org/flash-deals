# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY services/product/go/go.mod services/product/go/go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/product/go/ ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary
COPY --from=builder /server .

# Expose ports
EXPOSE 8002 50051

# Run
CMD ["./server"]
