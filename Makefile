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
