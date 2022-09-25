.PHONY: build-run build clean

build-run:
	make build
	./target/linux/valheim-launcher

build:
	go build -o target/linux/valheim-launcher main.go
	cp config/config.toml target/linux/

clean:
	rm -rf target

bundle-font:
	fyne bundle --package fonts --prefix Resource --name DefaultFont -o fonts/default_font.go <font-file>
	#fyne bundle --package fonts --prefix Resource --name DefaultFont -o fonts/default_font.go ~/.local/share/fonts/HarmonyOS_Sans_SC_Regular.ttf

bundle-font-build:
	fyne bundle --package fonts --prefix Resource --name DefaultFont -o fonts/default_font.go /usr/local/share/fonts/HarmonyOS_Sans_SC_Regular.ttf
