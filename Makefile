windows_arm:
	GOOS=windows GOARCH=arm go build -o builds/fileserver_win_arm.exe .

windows_amd:
	GOOS=windows GOARCH=amd64 go build -o builds/fileserver_win_amd64.exe .

mac:
	go build -o builds/fileserver_mac .