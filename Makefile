test:
	@go test ./...

build:
	@go build -o wildgecu .

lint:
	@mise exec -- golangci-lint run ./...
