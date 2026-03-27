CONTAINER_NAME = clink_dev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-X main.Version=$(VERSION)"

run:
	go run *.go

build:
	go build $(LDFLAGS) -o clink .

test:
	go test github.com/alexmaze/clink/...

# 发布多平台二进制（供 GoReleaser 或手动使用）
build-all:
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/clink-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/clink-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/clink-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/clink-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/clink-windows-amd64.exe .

# make crun args="-c .test/config.yaml"
crun:
	docker exec \
		-i -t \
		$(CONTAINER_NAME) \
		go run *.go ${args}

cbash:
	docker exec -i -t $(CONTAINER_NAME) /bin/bash

ccreate:
	docker run \
		-t -d \
		-h dev \
		--name $(CONTAINER_NAME) \
		-v $(shell pwd):/clink \
		-w /clink \
		golang:1.18 \

crm:
	docker rm -f $(CONTAINER_NAME)
