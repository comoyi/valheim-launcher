.PHONY: build-run build

build-run:
	make build
	./target/linux/valheim-launcher

build:
	go build -o target/linux/valheim-launcher main.go
	cp config/config.toml target/linux/
