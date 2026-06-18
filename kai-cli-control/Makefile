.PHONY: build clean install vet

BINARY=kaictl

build:
	go build -o $(BINARY) .

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

install:
	go install .
