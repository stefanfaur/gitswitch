BIN := gitswitch
GOFLAGS := -trimpath -ldflags="-s -w"

.PHONY: build test vet fmt clean

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BIN) ./cmd/gitswitch

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

clean:
	rm -f $(BIN)

# --- release helpers (local testing; CI is source of truth) ---

RELEASE_OS_ARCHES := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: release-local checksums

release-local:
	@set -eu; \
	rm -rf dist && mkdir -p dist; \
	VER=$${VER:-dev-local}; \
	for pair in $(RELEASE_OS_ARCHES); do \
	  os=$${pair%/*}; arch=$${pair#*/}; \
	  name="gitswitch-$$VER-$$os-$$arch"; \
	  mkdir -p "dist/$$name"; \
	  echo "building $$name ..."; \
	  GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
	    go build -trimpath \
	      -ldflags="-s -w -X main.version=$$VER" \
	      -o "dist/$$name/gitswitch" ./cmd/gitswitch; \
	  ( cd dist && tar -czf "$$name.tar.gz" "$$name" && rm -rf "$$name" ); \
	done

checksums:
	@set -eu; cd dist; \
	if command -v sha256sum >/dev/null; then sha256sum gitswitch-*.tar.gz > SHA256SUMS; \
	else shasum -a 256 gitswitch-*.tar.gz > SHA256SUMS; fi; \
	echo "wrote dist/SHA256SUMS"
