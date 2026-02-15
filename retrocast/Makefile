.PHONY: build test run clean

build:
	go build -o bin/retrocast ./cmd/retrocast

test:
	go test -v -race ./...

run: build
	./bin/retrocast

clean:
	rm -rf bin/
