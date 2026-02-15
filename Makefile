.PHONY: build install clean

# Build locally
build:
	go build -o agm.exe ./cmd/agm

# Install to $GOPATH/bin (available globally)
install:
	go install ./cmd/agm

# Clean build artifacts
clean:
	rm -f agm agm.exe
