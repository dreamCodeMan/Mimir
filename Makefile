.PHONY: build

build:
	@go build

run: build
	./Mimir