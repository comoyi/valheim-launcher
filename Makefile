.PHONY: build-run build clean

X_APP_VERSION := $(shell cat VERSION)

build-run:
	make build
	./target/linux/valheim-launcher

build:
	go build -o target/linux/valheim-launcher main.go
	cp config/config.toml target/linux/

package-linux:
	make build
	cd target/linux && tar zcvf valheim-launcher-$(X_APP_VERSION)-linux.tar.gz config.toml valheim-launcher && cd -

package-linux-installer:
	fyne package -os linux --release
	mkdir -p target/linux
	mv valheim-launcher.tar.xz target/linux/valheim-launcher-$(X_APP_VERSION)-linux-installer.tar.xz

package-windows:
	mkdir -p target/windows
	CC=x86_64-w64-mingw32-gcc fyne package -os windows --release --appID com.comoyi.valheim-launcher --name target/windows/valheim-launcher.exe
	cp config/config.toml target/windows/
	cd target/windows && zip valheim-launcher-$(X_APP_VERSION)-windows.zip config.toml valheim-launcher.exe && cd -

clean:
	rm -rf target

bundle-font:
	fyne bundle --package fonts --prefix Resource --name DefaultFont -o fonts/default_font.go <font-file>
	#fyne bundle --package fonts --prefix Resource --name DefaultFont -o fonts/default_font.go ~/.local/share/fonts/HarmonyOS_Sans_SC_Regular.ttf

bundle-font-build:
	fyne bundle --package fonts --prefix Resource --name DefaultFont -o fonts/default_font.go /usr/local/share/fonts/HarmonyOS_Sans_SC_Regular.ttf

deps:
	go get fyne.io/fyne/v2
	go install fyne.io/fyne/v2/cmd/fyne@latest
