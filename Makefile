CONTAINER_NAME = clink_dev

run:
	go run *.go 

test:
	go test github.com/alexmaze/clink/...

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
