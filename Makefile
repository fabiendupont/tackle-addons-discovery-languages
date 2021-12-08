GOBIN ?= ${GOPATH}/bin

all: addon-adapter

fmt:
	go fmt ./...

vet:
	go vet ./...

addon-adapter: fmt vet
	go build -ldflags="-w -s" -o bin/addon-adapter github.com/konveyor/tackle-addons-discovery-languages
