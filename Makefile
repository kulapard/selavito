VERSION = 1.0.0

all: mac32 mac64 linux32 linux64 win32 win64

mac32:
	GOOS=darwin GOARCH=386 go build -o _build/selavito_$(VERSION)_darwin_i386/selavito
	cd _build && tar -cvzf selavito_$(VERSION)_darwin_i386.tar.gz selavito_$(VERSION)_darwin_i386/selavito
	rm -rf _build/selavito_$(VERSION)_darwin_i386

mac64:
	GOOS=darwin GOARCH=amd64 go build -o _build/selavito_$(VERSION)_darwin_amd64/selavito
	cd _build && tar -cvzf selavito_$(VERSION)_darwin_amd64.tar.gz selavito_$(VERSION)_darwin_amd64/selavito
	rm -rf _build/selavito_$(VERSION)_darwin_amd64

linux32:
	GOOS=linux GOARCH=386 go build -o _build/selavito_$(VERSION)_linux_i386/selavito
	cd _build && tar -cvzf selavito_$(VERSION)_linux_i386.tar.gz selavito_$(VERSION)_linux_i386/selavito
	rm -rf _build/selavito_$(VERSION)_linux_i386

linux64:
	GOOS=linux GOARCH=amd64 go build -o _build/selavito_$(VERSION)_linux_amd64/selavito
	cd _build && tar -cvzf selavito_$(VERSION)_linux_amd64.tar.gz selavito_$(VERSION)_linux_amd64/selavito
	rm -rf _build/selavito_$(VERSION)_linux_amd64

win32:
	GOOS=windows GOARCH=386 go build -o _build/selavito_$(VERSION)_windows_i386/selavito.exe
	cd _build && zip selavito_$(VERSION)_windows_i386.zip selavito_$(VERSION)_windows_i386/selavito.exe
	rm -rf _build/selavito_$(VERSION)_windows_i386

win64:
	GOOS=windows GOARCH=amd64 go build -o _build/selavito_$(VERSION)_windows_amd64/selavito.exe
	cd _build && zip selavito_$(VERSION)_windows_amd64.zip selavito_$(VERSION)_windows_amd64/selavito.exe
	rm -rf _build/selavito_$(VERSION)_windows_amd64
