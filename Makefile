default: clean check build test

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

clean:
	rm -rf ./uncors ./uncors.exe coverage.out
	rm -rf ./tools/fakedata/docs.md ./tools/fakedata/scheme.json

check:
	make format
	make test
	make build
