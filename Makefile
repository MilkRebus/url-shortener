.PHONY: run test race vet fmt build docker-up docker-down

run:
	go run ./cmd/shortener

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find cmd internal -name '*.go')

build:
	mkdir -p bin
	go build -o bin/url-shortener ./cmd/shortener

docker-up:
	docker compose up --build

docker-down:
	docker compose down
