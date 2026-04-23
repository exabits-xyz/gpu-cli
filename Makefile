default: build

-include .env
export

BINARY        := egpu
MODULE        := github.com/exabits-xyz/gpu-cli
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS       := -ldflags "-X $(MODULE)/cmd.Version=$(VERSION)"
GITHUB_TOKEN  ?= $(shell gh auth token 2>/dev/null)

.PHONY: default build install clean test fmt vet release upload npm-publish

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

clean:
	rm -f $(BINARY) $(BINARY).exe
	rm -rf dist/

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

release:
	@$(if $(strip $(version)),:,$(error version is required: make release version=v1.2.3))
	git tag $(version) && git push origin $(version)
	goreleaser release --clean

upload:
	@$(if $(strip $(version)),:,$(error version is required: make upload version=v1.2.3))
	gh release create $(version) ./dist/*.tar.gz ./dist/*.zip ./dist/checksums.txt \
		--generate-notes --title "$(version)" --latest

npm-publish:
	npm publish --access public
