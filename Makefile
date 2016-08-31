all: build

test:
	go test -v -cover ./...

lint:
	go vet ./...

deps:
	go get -d -t ./...

build: test lint clean
	GOOS=darwin GOARCH=amd64 go build -o build/ec2-filter_darwin_amd64
	GOOS=linux  GOARCH=amd64 go build -o build/ec2-filter_linux_amd64
	cd build && shasum -a256 ec2-filter_* > SHA256SUMS

clean:
	$(RM) -r build

.PHONY: build
