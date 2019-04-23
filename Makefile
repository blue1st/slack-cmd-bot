default: build

run:
	go run main.go -c config.yml

build:
	go build

build-mac:
	GOOS=darwin GOARCH=amd64 go build

build-win:
	GOOS=windows GOARCH=amd64 go build

build-linux:
	GOOS=linux GOARCH=amd64 go build

.PHONY: run build build-mac build-win build-linux
