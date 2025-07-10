format:
	@gofumpt -l -w .

test:
	@go test -v ./...
