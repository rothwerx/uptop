docker-build:
	docker run --rm -v $(shell pwd):/work -w /work golang:latest go build -o uptop .
