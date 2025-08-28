# --- Build Stage ---
FROM golang:latest AS builder

WORKDIR /app

# Install build dependencies
# RUN apk add --no-cache git

# Cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .


# Build statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -extldflags '-static'" -o /syac

# --- Runtime Stage ---
FROM alpine:3.21

# Install only runtime dependencies
RUN apk add --no-cache docker-cli


# Copy binary from build stage
COPY --from=builder /syac /usr/local/bin/syac
