default: setup build 

setup:
	go get -v -u \
		github.com/laher/goxc \
		github.com/tcnksm/ghr

run:
	go run main.go -c config.yml

build:
	go build

deploy:
	script/release

.PHONY: run 
