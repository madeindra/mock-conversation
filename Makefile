.PHONY: build build-server build-client run run-server run-client install clean stop

# Build both server and client
build: build-server build-client

# Build server binary
build-server:
	cd server && go build -o ../bin/server .

# Build client static files
build-client:
	cd client && npm run build

# Run both server (background) and client (foreground)
run:
	cd server && go run main.go & echo $$! > .server.pid
	cd client && npm run dev
	@$(MAKE) stop

# Run server only
run-server:
	cd server && go run main.go

# Run client dev server only
run-client:
	cd client && npm run dev

# Stop background server
stop:
	@if [ -f .server.pid ]; then kill $$(cat .server.pid) 2>/dev/null; rm -f .server.pid; echo "Server stopped"; fi

# Install client dependencies
install:
	cd client && npm install

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf client/dist/
