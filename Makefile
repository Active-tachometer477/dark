.PHONY: test build dev lint fmt example-db clean

test:
	go test ./...

build:
	go build ./...

dev:
	cd _examples/hello && go run main.go

lint:
	go vet ./...

fmt:
	gofmt -w .

example-db:
	cd _examples/database && go run main.go

clean:
	go clean -cache
