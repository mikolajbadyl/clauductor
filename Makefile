APP_NAME := clauductor
BACKEND_DIR := backend
FRONTEND_DIR := $(BACKEND_DIR)/frontend
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.buildVersion=$(VERSION)"

.PHONY: all clean dev build frontend backend cross

all: build

# Development: run frontend and backend separately
dev:
	@echo "Start frontend: npm run dev"
	@echo "Start backend:  cd backend && go run ."

# Build production single binary
build: frontend backend

frontend:
	npm run generate
	rm -rf $(FRONTEND_DIR)
	cp -r .output/public $(FRONTEND_DIR)

backend:
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build $(LDFLAGS) -o ../$(APP_NAME) .

# Cross-compile for all platforms
cross: frontend
	@mkdir -p dist
	cd $(BACKEND_DIR) && GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o ../dist/$(APP_NAME)-linux-amd64 .
	cd $(BACKEND_DIR) && GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o ../dist/$(APP_NAME)-linux-arm64 .
	cd $(BACKEND_DIR) && GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o ../dist/$(APP_NAME)-darwin-amd64 .
	cd $(BACKEND_DIR) && GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o ../dist/$(APP_NAME)-darwin-arm64 .
	cd $(BACKEND_DIR) && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o ../dist/$(APP_NAME)-windows-amd64.exe .
	@echo "Binaries in dist/"

clean:
	rm -rf $(FRONTEND_DIR) .output dist $(APP_NAME)
