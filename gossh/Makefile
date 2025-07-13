
.PHONY: clean all

all: linux-amd64 linux-arm64 macos-amd64 macos-arm64 windows-amd64

linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o WebSSH-linux-amd64

linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o WebSSH-linux-arm64

macos-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o WebSSH-macos-amd64

macos-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o WebSSH-macos-arm64

windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o WebSSH-windows-amd64.exe

WebSSH:
	go build -o WebSSH

clean:
	rm -f WebSSH-linux-amd64 WebSSH-linux-arm64 WebSSH-macos-amd64 WebSSH-macos-arm64 WebSSH-windows-amd64.exe
