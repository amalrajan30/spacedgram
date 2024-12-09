.PHONY: build run

build:
	go build -o bin/spacedgram ./cmd/bot

run:
	go run ./cmd/bot