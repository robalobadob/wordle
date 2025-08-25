# apps/go-server/Dockerfile
#
# Multi-stage Dockerfile for the Go server.
# Produces a small final image containing only the compiled binary.
#
# Stage 1: Builder
#   • Uses golang:1.22-alpine (lightweight Go environment)
#   • Copies source and builds the binary
#
# Stage 2: Runtime
#   • Based on plain Alpine (no Go toolchain)
#   • Copies compiled binary from builder stage
#   • Sets working directory and environment
#   • Exposes port 5175 and runs the server
#
# This keeps the final image minimal (~20MB vs hundreds).

# ---- Build stage ----
FROM golang:1.22-alpine as builder

# Set working directory inside container
WORKDIR /app

# Copy source code into the container
COPY . .

# Build the go-server binary
RUN go build -o go-server .

# ---- Runtime stage ----
FROM alpine:3.19

# Set runtime working directory
WORKDIR /app

# Copy only the built binary from builder
COPY --from=builder /app/go-server /usr/local/bin/go-server

# Expose app port (matches PORT env var)
EXPOSE 5175

# Default environment variable (can be overridden at runtime)
ENV PORT=5175

# Start the server
CMD ["go-server"]
