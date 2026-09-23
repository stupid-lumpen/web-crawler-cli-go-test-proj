.PHONY: build run test lint clean

BINARY_NAME=crawler
CMD_PATH=./cmd/crawler

build:
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)

run:
	go run $(CMD_PATH)

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
