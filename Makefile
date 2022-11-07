CONFIG_PATH=${HOME}/.golog

.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY: test
test: $(CONFIG_PATH)/policy.csv $(CONFIG_PATH)/model.conf
	go test -race ./...

TAG ?= 0.0.1

build-docker:
	docker build -t github.com/denisschmidt/golog:$(TAG) .
