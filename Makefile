CONTAINER_NAME = clink_dev

run:
	go run *.go 

test:
	go test github.com/alexmaze/clink/...

c_run:
	docker exec \
		-i -t \
		$(CONTAINER_NAME) \
		go run *.go

c_bash:
	docker exec -i -t $(CONTAINER_NAME) /bin/bash

c_create:
	docker run \
		-t -d \
		-h dev \
		--name $(CONTAINER_NAME) \
		-v $(shell pwd):/clink \
		-w /clink \
		golang:1.17 \

c_rm:
	docker rm -f $(CONTAINER_NAME)
