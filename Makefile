.PHONY: build build-backend build-web run clean

# Build both backend and frontend
build: build-backend build-web

# Build backend only
build-backend:
	@echo "Building backend..."
	cd cmd/jarvis && go build -o jarvis jarvis.go
	@echo "Backend build complete: cmd/jarvis/jarvis"

# Build frontend only
build-web:
	@echo "Building frontend..."
	cd web && npm install && npm run build
	@echo "Frontend build complete: web/dist"

# Run the backend server
run:
	cd cmd/jarvis && ./jarvis -f etc/jarvis.yaml

# Clean build artifacts
clean:
	rm -f cmd/jarvis/jarvis
	rm -rf web/dist
	rm -rf web/node_modules

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing frontend dependencies..."
	cd web && npm install
