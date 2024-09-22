#!/bin/bash

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to prompt for installation
prompt_install() {
    read -p "Would you like to install $1? (y/n) " choice
    case "$choice" in 
      y|Y ) return 0;;
      n|N ) return 1;;
      * ) echo "Invalid input. Please enter y or n."; prompt_install "$1";;
    esac
}

# Function to install gum
install_gum() {
    echo "Installing gum..."
    if command_exists "brew"; then
        brew install gum
    elif command_exists "apt-get"; then
        sudo mkdir -p /etc/apt/keyrings
        curl -fsSL https://repo.charm.sh/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/charm.gpg
        echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | sudo tee /etc/apt/sources.list.d/charm.list
        sudo apt update && sudo apt install gum
    elif command_exists "yum"; then
        echo "[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key" | sudo tee /etc/yum.repos.d/charm.repo
        sudo yum install gum
    else
        echo "Unable to install gum. Please install it manually: https://github.com/charmbracelet/gum#installation"
        exit 1
    fi
}

# Function to install protoc
install_protoc() {
    echo "Installing protoc..."
    if command_exists "brew"; then
        brew install protobuf
    elif command_exists "apt-get"; then
        sudo apt-get update
        sudo apt-get install -y protobuf-compiler
    elif command_exists "yum"; then
        sudo yum install -y protobuf-compiler
    else
        echo "Unable to install protoc. Please install it manually: https://grpc.io/docs/protoc-installation/"
        exit 1
    fi
}

# Function to update proto file content
update_proto_file() {
    local proto_file="$1"
    log "Updating $proto_file..."

    # Update the file content
    cat > "$proto_file" <<EOL
syntax = "proto3";

package gitspace.plugin;

option go_package = "github.com/ssotops/gitspace-plugin-sdk/proto";

message PluginInfo {
    string name = 1;
    string version = 2;
}

message PluginInfoRequest {}

message CommandRequest {
    string command = 1;
    map<string, string> parameters = 2;
}

message CommandResponse {
    bool success = 1;
    string result = 2;
    string error_message = 3;
}

message MenuRequest {}

message MenuItem {
    string label = 1;
    string command = 2;
}

message MenuResponse {
    repeated MenuItem items = 1;
}

service PluginService {
    rpc GetPluginInfo(PluginInfoRequest) returns (PluginInfo) {}
    rpc ExecuteCommand(CommandRequest) returns (CommandResponse) {}
    rpc GetMenu(MenuRequest) returns (MenuResponse) {}
}
EOL

    success "$proto_file updated successfully"
    changes+=("Updated $proto_file")
}

# Check and install gum if necessary
if ! command_exists gum; then
    echo "gum is not installed."
    if prompt_install "gum"; then
        install_gum
    else
        echo "gum is required for this script. Exiting."
        exit 1
    fi
fi

# Check and install protoc if necessary
if ! command_exists protoc; then
    echo "protoc is not installed."
    if prompt_install "protoc"; then
        install_protoc
    else
        echo "protoc is required for this script. Exiting."
        exit 1
    fi
fi

# Function to print styled header
print_header() {
    gum style \
        --foreground 212 --border-foreground 212 --border double \
        --align center --width 50 --margin "1 2" --padding "2 4" \
        'Gitspace Plugin SDK Builder'
}

# Function to print styled log message
log() {
    gum style --foreground 39 "$(gum style --bold "➡")" "$1"
}

# Function to print styled success message
success() {
    gum style --foreground 76 "$(gum style --bold "✓")" "$1"
}

# Function to print styled error message
error() {
    gum style --foreground 196 "$(gum style --bold "✗")" "$1" >&2
}

# Function to install protoc-gen-go and protoc-gen-go-grpc
install_protoc_gen_go() {
    log "Installing protoc-gen-go and protoc-gen-go-grpc..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    success "protoc-gen-go and protoc-gen-go-grpc installed."
}

# Function to add local module replacement
add_local_module_replacement() {
    log "Adding local module replacement..."
    go mod edit -replace github.com/ssotops/gitspace-plugin-sdk=./
    if [ $? -eq 0 ]; then
        success "Local module replacement added successfully."
        changes+=("Local module replacement added")
    else
        error "Failed to add local module replacement."
        exit 1
    fi
}

# Function to ensure proto package is properly set up
ensure_proto_package() {
    log "Ensuring proto package is set up..."
    if [ ! -f "proto/plugin.go" ]; then
        echo "package proto" > proto/plugin.go
        success "Created proto/plugin.go"
        changes+=("Created proto/plugin.go")
    fi
}

# Print header
print_header

# Initialize variables to track changes
changes=()

# Install protoc-gen-go and protoc-gen-go-grpc
install_protoc_gen_go

# Check and create proto directory if it doesn't exist
if [ ! -d "proto" ]; then
    log "Creating proto directory..."
    mkdir -p proto
    success "Proto directory created."
    changes+=("Proto directory created")
fi

# Check if plugin.proto exists, if not create it, otherwise update it
proto_file="proto/plugin.proto"
if [ ! -f "$proto_file" ]; then
    log "No plugin.proto file found. Creating a new one..."
    update_proto_file "$proto_file"
    changes+=("Created new plugin.proto file")
else
    log "Existing plugin.proto file found. Updating..."
    update_proto_file "$proto_file"
fi

# Ensure proto package is set up
ensure_proto_package

# Generate protobuf files
log "Generating Go files from .proto definitions..."
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/*.proto
if [ $? -eq 0 ]; then
    success "Go files generated successfully from .proto definitions."
    changes+=("Proto Go files generated")
else
    error "Failed to generate Go files from .proto definitions."
    exit 1
fi

# Add local module replacement
add_local_module_replacement

# Update go.mod with required dependencies
log "Updating go.mod with required dependencies..."
go get google.golang.org/protobuf
go get google.golang.org/grpc
if [ $? -eq 0 ]; then
    success "Dependencies added to go.mod successfully."
    changes+=("Dependencies added to go.mod")
else
    error "Failed to add dependencies to go.mod."
    exit 1
fi

# Tidy up the go.mod file
log "Tidying up go.mod..."
go mod tidy
if [ $? -eq 0 ]; then
    success "go.mod tidied successfully."
    changes+=("go.mod tidied")
else
    error "Failed to tidy go.mod."
    exit 1
fi

# Update dependencies
log "Updating dependencies..."
go get -u ./...
if [ $? -eq 0 ]; then
    success "Dependencies updated successfully."
    changes+=("Dependencies updated")
else
    error "Failed to update dependencies."
    exit 1
fi

# Build the project
log "Building the project..."
go build ./...
if [ $? -eq 0 ]; then
    success "Project built successfully."
    changes+=("Project built")
else
    error "Failed to build the project."
    exit 1
fi

# Run tests
log "Running tests..."
go test ./...
if [ $? -eq 0 ]; then
    success "All tests passed."
    changes+=("Tests passed")
else
    error "Some tests failed."
    exit 1
fi

# Build example plugins
log "Building example plugins..."
(
    cd examples
    ./build.sh
)
if [ $? -eq 0 ]; then
    success "Example plugins built successfully."
    changes+=("Example plugins built")
else
    error "Failed to build example plugins."
    exit 1
fi

# Print summary
gum style \
    --foreground 226 --border-foreground 226 --border normal \
    --align left --width 50 --margin "1 2" --padding "1 2" \
    "Summary of Changes:"

for change in "${changes[@]}"; do
    gum style --foreground 226 "• $change"
done

success "Build process completed successfully!"
