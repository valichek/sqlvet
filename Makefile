GIT_COMMIT := $(shell git rev-parse --short HEAD)

test:
	go mod tidy
	go test -failfast -timeout 20s -race ./...

install:
	go install -ldflags "-X main.gitCommit=$(GIT_COMMIT)" .

cover:
	go test -coverprofile=go-cover.profile -timeout 5s ./...
	go tool cover -html=go-cover.profile
	rm go-cover.profile
