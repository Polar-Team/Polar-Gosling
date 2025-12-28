FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Gosling CLI binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gosling ./cmd/gosling

# Final stage - minimal runtime image
FROM alpine:latest

# Install required runtime dependencies
RUN apk --no-cache add ca-certificates git curl bash

# Copy Gosling CLI binary from builder
COPY --from=builder /app/gosling /usr/local/bin/gosling
RUN chmod +x /usr/local/bin/gosling

# Pre-install GitLab Runner Agent
# Note: This will be downloaded at build time from GitLab's official repository
ADD https://gitlab-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-runner-linux-amd64 /usr/local/bin/gitlab-runner
RUN chmod +x /usr/local/bin/gitlab-runner

# Create directory for OpenTofu binaries (mounted from S3 at runtime)
RUN mkdir -p /mnt/tofu_binary

# Set working directory
WORKDIR /workspace

# Set entrypoint to Gosling runner mode
ENTRYPOINT ["/usr/local/bin/gosling", "runner"]
