VERSION ?= "dev"
SOURCE_URL ?= ""

.PHONY: generate
generate: go.anx.io
	@rm -rf public
	./go.anx.io --mode generate

.PHONY: serve
serve: go.anx.io
	./go.anx.io --mode serve

.PHONY: go.anx.io
go.anx.io:
	go build -ldflags "-X main.version=$(VERSION) -X main.sourceURL=$(SOURCE_URL)" -o go.anx.io cmd/main.go
