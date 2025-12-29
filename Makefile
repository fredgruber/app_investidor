.PHONY: run build clean test

run:
	go run main.go

build:
	go build -o dca-app main.go

test:
	go test ./...

clean:
	rm -f dca-app
