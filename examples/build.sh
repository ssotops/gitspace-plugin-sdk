#!/bin/bash

# Function to print styled log message
log() {
    echo "➡ $1"
}

# Function to print styled success message
success() {
    echo "✓ $1"
}

# Function to print styled error message
error() {
    echo "✗ $1" >&2
}

# Function to set up plugin dependencies
setup_plugin_dependencies() {
    local plugin_dir="$1"
    (
        cd "$plugin_dir"
        go mod edit -replace github.com/ssotops/gitspace-plugin-sdk=../..
        go get github.com/ssotops/gitspace-plugin-sdk@latest
        go get github.com/charmbracelet/huh@latest
        go mod tidy
    )
}

# Build all plugins in the examples directory
build_plugins() {
    for plugin_dir in */; do
        if [ -d "$plugin_dir" ]; then
            log "Setting up dependencies for plugin: ${plugin_dir%/}"
            setup_plugin_dependencies "$plugin_dir"
            
            log "Building plugin: ${plugin_dir%/}"
            (
                cd "$plugin_dir"
                go build -o "${plugin_dir%/}"
            )
            if [ $? -eq 0 ]; then
                success "Plugin ${plugin_dir%/} built successfully."
            else
                error "Failed to build plugin ${plugin_dir%/}."
                exit 1
            fi
        fi
    done
}

# Main execution
log "Building example plugins..."
build_plugins
success "All example plugins built successfully."
