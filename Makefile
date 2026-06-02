.PHONY: test build clean

test:
go test ./...

build:
./scripts/build.sh

clean:
rm -rf dist
