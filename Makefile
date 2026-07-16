.PHONY: run build tidy vet

run:
	go run ./cmd/bot

build:
	go build -o bin/gojobs-bot ./cmd/bot

tidy:
	go mod tidy

vet:
	go vet ./...
