.PHONY: build run test clean tidy

build:
	go build -o bin/simulator ./cmd/simulator

run:
	go run ./cmd/simulator

test:
	go test ./...

clean:
	rm -rf bin/ output/

tidy:
	go mod tidy
