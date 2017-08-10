.PHONY: build

build:
	export TAG=dev-`date +%Y%m%d`
	go build -ldflags "-X main._VERSION_=`echo $(TAG)`"

run: build
	./Mimir