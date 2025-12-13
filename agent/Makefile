.PHONY: build clean install

# Binary name
BINARY=watchflare-agent

# Build the agent
build:
	go build -o $(BINARY) main.go

# Build for Linux (useful for cross-compilation)
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux main.go

# Clean build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-linux

# Install the agent (requires sudo)
install: build
	sudo mkdir -p /usr/local/bin
	sudo cp $(BINARY) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY)
	@echo "Agent installed to /usr/local/bin/$(BINARY)"

# Uninstall the agent
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY)
	sudo rm -rf /etc/watchflare
	@echo "Agent uninstalled"
