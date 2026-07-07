# TaskMax — Makefile
# Cosy Pomodoro task manager (Go + Wails v2 + Svelte)
#
# Quick start:
#   make setup        # install all dependencies (Go + frontend + Wails CLI)
#   make system-deps  # Linux/WSL ONLY: install GTK + WebKit libs (needs sudo)
#   make dev          # run the app with hot-reload (recommended for development)
#   make build        # produce a distributable binary in ./build/bin
#   make run          # build, then launch the binary
#
# On native Windows/macOS you do NOT need `make system-deps` — the OS webview
# (WebView2 / WebKit) is already present. Just: make setup && make dev

# Resolve the Wails CLI: prefer one on PATH, else fall back to $GOPATH/bin.
GOPATH      := $(shell go env GOPATH)
WAILS       := $(shell command -v wails 2>/dev/null || echo $(GOPATH)/bin/wails)
FRONTEND    := frontend

# Binary name differs per OS; Wails handles the extension automatically.
APP_NAME    := TaskMax

# On Linux, Wails needs a build tag matching the installed WebKit version.
# This box has webkit2gtk-4.0, so we pass `-tags webkit2_40`. Override if you
# have 4.1 installed:  make dev WAILS_TAGS=webkit2_41
# On Windows/macOS the tag is unused (WebView2 / WebKit are built in).
UNAME_S     := $(shell uname -s 2>/dev/null)
ifeq ($(UNAME_S),Linux)
WAILS_TAGS  ?= webkit2_40
endif
TAGFLAG     := $(if $(WAILS_TAGS),-tags $(WAILS_TAGS),)

.DEFAULT_GOAL := help

## ----------------------------------------------------------------------------
## Help
## ----------------------------------------------------------------------------
.PHONY: help
help: ## Show this help
	@echo "TaskMax — available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "} {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
	@echo ""

## ----------------------------------------------------------------------------
## Setup
## ----------------------------------------------------------------------------
.PHONY: setup
setup: install-wails deps ## Install the Wails CLI and all project dependencies

.PHONY: install-wails
install-wails: ## Install/upgrade the Wails v2 CLI
	go install github.com/wailsapp/wails/v2/cmd/wails@latest
	@echo "Wails installed to $(GOPATH)/bin — make sure that dir is on your PATH."

.PHONY: deps
deps: ## Download Go modules and install frontend npm packages
	go mod tidy
	cd $(FRONTEND) && npm install

.PHONY: doctor
doctor: ## Check that the toolchain is ready (Wails, Go, Node)
	$(WAILS) doctor

.PHONY: system-deps
system-deps: ## (Linux/WSL only) Install the GTK + WebKit libraries Wails needs
	sudo apt update
	sudo apt install -y libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config build-essential

## ----------------------------------------------------------------------------
## Develop / Run
## ----------------------------------------------------------------------------
.PHONY: dev
dev: ## Run the app with hot-reload (Go + Svelte). Best way to "see it running".
	$(WAILS) dev $(TAGFLAG)

.PHONY: build
build: ## Build a production desktop binary into ./build/bin
	$(WAILS) build $(TAGFLAG)

.PHONY: build-debug
build-debug: ## Build with the debug console + devtools enabled
	$(WAILS) build -debug -devtools $(TAGFLAG)

.PHONY: run
run: build ## Build then launch the compiled binary
	@echo "Launching $(APP_NAME)..."
	@if [ -f "build/bin/$(APP_NAME).exe" ]; then \
		./build/bin/$(APP_NAME).exe ; \
	elif [ -f "build/bin/$(APP_NAME)" ]; then \
		./build/bin/$(APP_NAME) ; \
	elif [ -d "build/bin/$(APP_NAME).app" ]; then \
		open "build/bin/$(APP_NAME).app" ; \
	else \
		echo "No binary found in build/bin — run 'make build' first." ; exit 1 ; \
	fi

## ----------------------------------------------------------------------------
## Frontend
## ----------------------------------------------------------------------------
.PHONY: frontend
frontend: ## Build the Svelte frontend only (outputs to frontend/dist)
	cd $(FRONTEND) && npm run build

.PHONY: frontend-dev
frontend-dev: ## Run the Vite dev server standalone (UI only, no Go backend)
	cd $(FRONTEND) && npm run dev

## ----------------------------------------------------------------------------
## Quality checks
## ----------------------------------------------------------------------------
.PHONY: check
check: vet backend-build test ## Run vet, build the backend packages, and run tests

.PHONY: vet
vet: ## Run go vet across the whole module
	go vet ./...

.PHONY: backend-build
backend-build: ## Compile the backend packages (no webview/webkit needed)
	go build ./internal/...

.PHONY: test
test: ## Run Go tests
	go test ./... 2>&1

## ----------------------------------------------------------------------------
## Housekeeping
## ----------------------------------------------------------------------------
.PHONY: clean
clean: ## Remove build artifacts and the local database
	rm -rf build/bin
	rm -rf $(FRONTEND)/dist
	rm -f tasks.db

.PHONY: clean-all
clean-all: clean ## Also remove downloaded node_modules
	rm -rf $(FRONTEND)/node_modules
