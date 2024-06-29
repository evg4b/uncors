format:
	gofmt -l -s -w .
	gofumpt -l -w .
	golangci-lint run --fix

update_deps:
	go get -u ./...
	go mod tidy

test:
	go test ./...

test-cover:
	go test -tags release -timeout 1m -race -v -coverprofile=coverage.out ./...

build:
	go build ./...

build-release:
	go build -tags release ./...