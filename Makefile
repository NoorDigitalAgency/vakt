.PHONY: test build package clean

test:
go test ./...

build:
./scripts/build.sh

package:
./scripts/package-release.sh

clean:
rm -rf dist
