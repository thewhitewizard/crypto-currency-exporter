.PHONY: build

default:build

build: tidy	
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/crypto-currency-exporter.amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64  go build -o bin/crypto-currency-exporter.arm64 .
tidy:
	go mod tidy

format:
	go fmt ./...

image: 
	docker build -t crypto-currency-exporter .


gomod_tidy:
	go mod tidy
