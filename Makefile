.PHONY: build format lint test check

build:
	@go build -o toggl-cli .

format:
	@gofumpt -l -w .

lint:
	@golangci-lint run ./...

test:
	@go test -race ./...

check: format lint test
