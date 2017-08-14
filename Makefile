.PHONY: build

build:
	go build -ldflags "-X main._VERSION_=$(shell date +%Y%m%d)"

run: build
	./Mimir