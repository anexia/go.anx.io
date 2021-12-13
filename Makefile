.PHONY: generate
generate: go.anx.io
	@rm -rf public
	./go.anx.io --mode generate

.PHONY: go.anx.io
go.anx.io:
	go build
