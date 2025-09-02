# GOTTH System Monitor Makefile

.PHONY: install-templ generate build run clean dev watch help

# Default target
help:
	@echo "🚀 GOTTH System Monitor - Available commands:"
	@echo "  install-templ  - Install templ CLI tool"
	@echo "  generate      - Generate Go code from templ templates"
	@echo "  build         - Build the application"
	@echo "  run           - Generate templates and run the application"
	@echo "  dev           - Development mode with auto-restart"
	@echo "  watch         - Watch for template changes and regenerate"
	@echo "  clean         - Clean generated files"

# Install templ CLI tool
install-templ:
	@echo "📦 Installing templ CLI..."
	go install github.com/a-h/templ/cmd/templ@latest

# Generate Go code from templ templates
generate:
	@echo "🔄 Generating templates..."
	templ generate

# Build the application
build: generate
	@echo "🔨 Building application..."
	go build -o bin/monitor .

# Run the application
run: generate
	@echo "🚀 Running GOTTH System Monitor..."
	go run .

# Development mode with auto-restart using air (if installed)
dev: generate
	@if command -v air > /dev/null; then \
		echo "🔥 Starting development server with air..."; \
		air; \
	else \
		echo "⚠️  Air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "🚀 Running normally..."; \
		go run .; \
	fi

# Watch for template changes and regenerate
watch:
	@echo "👀 Watching for template changes..."
	templ generate --watch

# Clean generated files
clean:
	@echo "🧹 Cleaning up..."
	rm -rf bin/
	find . -name "*_templ.go" -delete

# Initialize project (run once)
init: install-templ
	@echo "🎉 Initializing GOTTH project..."
	go mod tidy
	@echo "✅ Project initialized! Run 'make run' to start."

# Production build
prod: generate
	@echo "📦 Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/monitor .

# Docker build (optional)
docker:
	@echo "🐳 Building Docker image..."
	docker build -t gotth-monitor .
