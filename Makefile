build:
	go build -o bin/envscan main.go

run:
	go run main.go

compile:
	echo "Compiling for every OS and Platform"
	GOOS=windows GOARCH=amd64 go build -o bin/envscan-windows.exe main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/envscan-darwin main.go
	GOOS=linux GOARCH=amd64 go build -o bin/envscan main.go

all: hello build