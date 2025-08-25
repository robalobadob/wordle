# apps/go-server/Makefile
#
# Convenience Makefile for common developer and CI tasks.
# Provides simple shortcuts for running, building, and containerizing the Go server.
#
# Targets:
#   run          – Run the Go server directly with `go run` (development mode).
#   build        – Compile the server binary into ./bin/go-server.
#   docker-build – Build a Docker image (tagged `wordle/go-server:dev`).
#   docker-run   – Run the Docker image with port 5175 exposed and env vars from .env.
#
# Usage:
#   make run
#   make build
#   make docker-build
#   make docker-run

# Run the server locally with the Go toolchain (development mode).
run:
	go run ./...

# Compile the binary into ./bin/go-server for local execution.
build:
	go build -o bin/go-server .

# Build the Docker image for the server, tagged as "wordle/go-server:dev".
docker-build:
	docker build -t wordle/go-server:dev .

# Run the server inside a container, mapping host port 5175 to container port 5175.
# Loads environment variables from .env for configuration.
docker-run:
	docker run --rm -p 5175:5175 --env-file .env wordle/go-server:dev
