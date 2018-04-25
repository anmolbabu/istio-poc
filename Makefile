DOCKER_NAME ?= osio-hdd/istio-poc
DOCKER_VERSION ?= latest
DOCKER_TAG = ${DOCKER_NAME}:${DOCKER_VERSION}

.PHONY: build docker-build

build:
	go build -o ${GOPATH}/bin/istio-poc
docker-build:
	go build -o ${GOPATH}/bin/istio-poc
	@mkdir -p _output/
	@cp ${GOPATH}/bin/istio-poc _output/
	@cp Dockerfile _output/
	@echo Building docker image into local docker daemon...
	docker build -t ${DOCKER_TAG} _output/
