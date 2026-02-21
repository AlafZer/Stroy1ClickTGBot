run-tests:
	SET GOROOT=C:\Program Files\Go
	SET GOPATH=C:\Users\vladi\go
	"C:\Program Files\Go\bin\go.exe" test -cover -v -coverprofile=coverage.out ./...
	go tool cover -html="coverage.out"
run:
	SET GOROOT=C:\Program Files\Go
	SET GOPATH=C:\Users\vladi\go
	GOOS=windows GOARCH=amd64 "C:\Program Files\Go\bin\go.exe" build -o C:\Users\vladi\AppData\Local\JetBrains\GoLand2025.3\tmp\GoLand\___1go_build_webapp.exe .