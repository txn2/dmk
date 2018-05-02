test:
	go test ./...

build:
	go build dmk.go

clean:
	rm ./dmk

.DEFAULT_GOAL := build