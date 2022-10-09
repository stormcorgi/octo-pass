all: mac_amd64 mac_arm64 win_amd64

mac_amd64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o ./build/octo-pass-amd64 ./main.go

mac_arm64:
	go build -o ./build/octo-pass-arm64 ./main.go

win_amd64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build -o ./build/octo-pass.exe ./main.go

archive: mac_amd64 mac_arm64 win_amd64
	zip octo-pass.zip -r bin -r build

clean:
	rm -f ./build/*