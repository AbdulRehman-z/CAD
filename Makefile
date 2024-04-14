build:
	go build -o bin/fs

run:
	go run main.go

test:
	go test ./... -v