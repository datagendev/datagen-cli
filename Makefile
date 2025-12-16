.PHONY: build install test clean release

# Build the binary
build:
	go build -o datagen

# Install to /usr/local/bin
install: build
	sudo mv datagen /usr/local/bin/

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f datagen
	rm -f datagen-*

# Build for multiple platforms
release:
	GOOS=darwin GOARCH=arm64 go build -o datagen-darwin-arm64
	GOOS=darwin GOARCH=amd64 go build -o datagen-darwin-amd64
	GOOS=linux GOARCH=amd64 go build -o datagen-linux-amd64
	GOOS=linux GOARCH=arm64 go build -o datagen-linux-arm64
	GOOS=windows GOARCH=amd64 go build -o datagen-windows-amd64.exe

# Development: build and run with example
dev: build
	./datagen --help
