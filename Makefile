VERSION ?= "dev"
SOURCE_URL ?= ""

generate: go.anx.io
	@rm -rf public
	./go.anx.io --mode generate

serve: go.anx.io
	./go.anx.io --mode serve

go.anx.io:
	go build -ldflags "-X main.version=$(VERSION) -X main.sourceURL=$(SOURCE_URL)" -o go.anx.io cmd/main.go

lint: tools
	tools/golangci-lint run ./...
	tools/misspell -error ./

tools:
	cd tools && go build -o . github.com/golangci/golangci-lint/cmd/golangci-lint
	cd tools && go build -o . github.com/client9/misspell/cmd/misspell

test: generate
	cd tools && go run test.go ../public ../packages.yaml

.PHONY: lint test go.anx.io serve generate tools
