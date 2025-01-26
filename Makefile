# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=twse_exporter

# Build the project
all: build

build:
	$(GOBUILD) -o $(BINARY_NAME) main.go

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) -v ./...

.PHONY: all build clean test