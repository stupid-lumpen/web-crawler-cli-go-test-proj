.PHONY: build run test lint clean

BINARY_NAME=crawler
CMD_PATH=./cmd/crawler

build:
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)

run:
	go run $(CMD_PATH)

test:
	go test -v -race ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-html: test-coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Отчет сохранен в coverage.html"

lint:
	golangci-lint run

clean:
	rm -rf bin/ coverage.out coverage.html result.json result.log
