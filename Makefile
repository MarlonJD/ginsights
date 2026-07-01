.PHONY: help test lint doctor build serve report clean

help:
	@echo "Targets:"
	@echo "  make test    - run Go tests"
	@echo "  make lint    - gofmt check + go vet"
	@echo "  make doctor  - validate docs/agent harness"
	@echo "  make build   - compile ginsights"
	@echo "  make serve   - serve this repo's local dashboard"
	@echo "  make report  - build static report into ./report"

test:
	go test ./...

lint:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || (echo "Run gofmt on changed Go files" && gofmt -l $$(find . -name '*.go' -not -path './.git/*') && exit 1)
	go vet ./...

doctor:
	go run ./cmd/ginsights doctor .

build:
	mkdir -p bin
	go build -o bin/ginsights ./cmd/ginsights

serve:
	go run ./cmd/ginsights serve . --port 43117

report:
	go run ./cmd/ginsights build . --out report

clean:
	rm -rf bin report .ginsights .ginsights-cache dist
