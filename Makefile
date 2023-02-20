NAME="pgbase"
VERSION=$(shell git describe --tags --always --first-parent 2>/dev/null)
COMMIT=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell date)

all: tidy test build

tidy:
	@echo "Tidy up..."
	@go mod tidy -v

test:
	@echo "Running tests..."
	@go test -cover ./...

build:
	@echo "Building..."
	@go build ./...
