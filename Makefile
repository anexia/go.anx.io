.PHONY: generate
generate: go.anx.io
	@rm -rf public
	./go.anx.io --mode generate

.PHONY: go.anx.io
go.anx.io:
	go build -o go.anx.io cmd/main.go
