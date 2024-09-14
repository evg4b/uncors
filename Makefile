default: test format build

test:
	go test -timeout 1m ./...

test-coverage:
	go test -tags release -timeout 1m -race -v -coverprofile=coverage.out ./...

format:
	gofumpt -l -w .
	gofmt -l -s -w .
	golangci-lint run --fix

build:
	go build -tags release -v .