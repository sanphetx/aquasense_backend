.PHONY: run build test swag tidy

# Run the server
run:
	go run cmd/server/main.go

# Build the binary
build:
	go build -o bin/aquasense-api cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Generate Swagger API documentation
swag:
	swag init -g cmd/server/main.go

# Tidy Go modules
tidy:
	go mod tidy
