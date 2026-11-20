check:
    go vet ./...
    golangci-lint run
    go test -race ./...
    actionlint
    if command -v zizmor >/dev/null; then zizmor --min-severity medium .; else nix run nixpkgs#zizmor -- --min-severity medium .; fi

calibrate dir model:
    go run ./cmd/strata calibrate --model {{model}} {{dir}}
