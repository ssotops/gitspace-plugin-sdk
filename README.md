# Gitspace Plugin SDK

The Gitspace Plugin SDK is a toolkit for developing plugins for the Gitspace application. It provides a set of tools and interfaces that allow developers to create custom plugins that can be seamlessly integrated into Gitspace.

## Table of Contents

## Table of Contents

1. [Installation](#installation)
2. [Usage](#usage)
3. [Plugin Structure](#plugin-structure)
4. [API Reference](#api-reference)
5. [Example Plugin](#example-plugin)
6. [Using Your Plugin with Gitspace](#using-your-plugin-with-gitspace)
   - [Installation](#installation-1)
   - [Running Your Plugin](#running-your-plugin)
   - [Plugin Location](#plugin-location)
   - [Troubleshooting](#troubleshooting)

## Installation

To use the Gitspace Plugin SDK in your project, run:

go get github.com/ssotops/gitspace-plugin-sdk

## Usage
To create a new plugin, you'll need to implement the PluginHandler interface provided by the SDK:

type PluginHandler interface {
    GetPluginInfo(*pb.PluginInfoRequest) (*pb.PluginInfo, error)
    ExecuteCommand(*pb.CommandRequest) (*pb.CommandResponse, error)
    GetMenu(*pb.MenuRequest) (*pb.MenuResponse, error)
}

## Plugin Structure
A typical plugin structure looks like this:

my-plugin/
├── main.go
└── gitspace-plugin.toml

The `gitspace-plugin.toml` file should contain metadata about your plugin:

```toml
[metadata]
name = "My Plugin"
version = "1.0.0"
description = "A sample plugin for Gitspace"

[[sources]]
path = "main.go"
entry_point = "Plugin"
```

## API Reference
- **GetPluginInfo**
  > This method should return information about your plugin, including its name and version.
- **ExecuteCommand**
  > This method is called when Gitspace wants to execute a command provided by your plugin.
- **GetMenu**
  > This method should return a menu structure that Gitspace will display to the user.

## Example Plugin
Here's a comprehensive example of a Hello World plugin implementation:

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/ssotops/gitspace-plugin-sdk/gsplug"
	"github.com/ssotops/gitspace-plugin-sdk/logger"
	pb "github.com/ssotops/gitspace-plugin-sdk/proto"
	"google.golang.org/protobuf/proto"
)

type HelloWorldPlugin struct{}

func (p *HelloWorldPlugin) GetPluginInfo(req *pb.PluginInfoRequest) (*pb.PluginInfo, error) {
	log.Info("GetPluginInfo called")
	return &pb.PluginInfo{
		Name:    "Hello World Plugin",
		Version: "1.0.0",
	}, nil
}

func (p *HelloWorldPlugin) ExecuteCommand(req *pb.CommandRequest) (*pb.CommandResponse, error) {
	switch req.Command {
	case "greet":
		name := req.Parameters["name"]
		if name == "" {
			name = "World"
		}
		return &pb.CommandResponse{
			Success: true,
			Result:  fmt.Sprintf("Hello, %s!", name),
		}, nil
	case "customize":
		greeting := req.Parameters["greeting"]
		name := req.Parameters["name"]
		if greeting == "" || name == "" {
			return &pb.CommandResponse{
				Success:      false,
				ErrorMessage: "Missing greeting or name parameter",
			}, nil
		}
		return &pb.CommandResponse{
			Success: true,
			Result:  fmt.Sprintf("%s, %s!", greeting, name),
		}, nil
	default:
		return &pb.CommandResponse{
			Success:      false,
			ErrorMessage: "Unknown command",
		}, nil
	}
}

func (p *HelloWorldPlugin) GetMenu(req *pb.MenuRequest) (*pb.MenuResponse, error) {
	menuOptions := []gsplug.MenuOption{
		{
			Label:   "Simple Greeting",
			Command: "greet",
			Parameters: []gsplug.ParameterInfo{
				{Name: "name", Description: "Name to greet", Required: false},
			},
		},
		{
			Label:   "Custom Greeting",
			Command: "customize",
			Parameters: []gsplug.ParameterInfo{
				{Name: "greeting", Description: "Custom greeting", Required: true},
				{Name: "name", Description: "Name to greet", Required: true},
			},
		},
	}

	menuBytes, err := json.Marshal(menuOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal menu: %w", err)
	}

	return &pb.MenuResponse{
		MenuData: menuBytes,
	}, nil
}

func main() {
	logDir := filepath.Join("logs", "hello-world")
	logger, err := logger.NewRateLimitedLogger(logDir, "hello-world")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Hello World plugin starting")

	plugin := &HelloWorldPlugin{}

	for {
		logger.Debug("Waiting for message")
		msgType, msg, err := gsplug.ReadMessage(os.Stdin)
		if err != nil {
			if err == io.EOF {
				logger.Info("Received EOF, exiting")
				return
			}
			logger.Error("Error reading message", "error", err)
			continue
		}
		logger.Debug("Received message", "type", msgType, "content", fmt.Sprintf("%+v", msg))

		var response proto.Message
		switch msgType {
		case 1: // GetPluginInfo
			response, err = plugin.GetPluginInfo(msg.(*pb.PluginInfoRequest))
		case 2: // ExecuteCommand
			response, err = plugin.ExecuteCommand(msg.(*pb.CommandRequest))
		case 3: // GetMenu
			response, err = plugin.GetMenu(msg.(*pb.MenuRequest))
		default:
			err = fmt.Errorf("unknown message type: %d", msgType)
		}

		if err != nil {
			logger.Error("Error handling message", "error", err)
			continue
		}

		logger.Debug("Sending response", "type", msgType, "content", fmt.Sprintf("%+v", response))
		err = gsplug.WriteMessage(os.Stdout, response)
		if err != nil {
			logger.Error("Error writing response", "error", err)
		} else {
			logger.Debug("Response sent successfully")
		}

		// Flush stdout to ensure the message is sent immediately
		os.Stdout.Sync()
	}
}
```

This example demonstrates a more complete plugin implementation, including:

- Proper error handling
- Logging
- Multiple commands with parameters
- A menu structure for the plugin's commands
- The main event loop for handling incoming messages

To use this plugin, you would also need to create a `gitspace-plugin.toml` file in the same directory:

```toml
[metadata]
name = "Hello World Plugin"
version = "1.0.0"
description = "A simple Hello World plugin for Gitspace"

[[sources]]
path = "main.go"
entry_point = "HelloWorldPlugin"
```

## Using Your Plugin with Gitspace

Once you've written plugin, you need to install and run it using Gitspace. Here's how:

### Installation

1. Build your plugin:
```sh
go build -o myplugin
```

2. Create a directory for your plugin in the Gitspace plugins folder:
```sh
mkdir -p ~/.ssot/gitspace/plugins/myplugin
```

3. Copy your built plugin and the `gitspace-plugin.toml` file to this directory:
```sh
cp myplugin ~/.ssot/gitspace/plugins/myplugin/
cp gitspace-plugin.toml ~/.ssot/gitspace/plugins/myplugin/
```

### Running Your Plugin

1. Start Gitspace:

> If you haven't already installed Gitspace locally, you can do so by following the instructions in the [Gitspace repository](https://github.com/ssotops/gitspace).

```sh
gitspace
```

2. In the Gitspace interface, you should now see your plugin listed in the available plugins.

3. Select your plugin to use its functionality.

### Plugin Location

Gitspace looks for plugins in the following directory:
```sh
~/.ssot/gitspace/plugins/
```

Each plugin should have its own subdirectory within this folder, containing the plugin binary and the `gitspace-plugin.toml` file.

### Troubleshooting

- If your plugin doesn't appear in Gitspace, ensure it's in the correct directory and that the `gitspace-plugin.toml` file is properly configured.
- Check Gitspace logs for any error messages related to plugin loading.
- Ensure your plugin has execute permissions: `chmod +x ~/.ssot/gitspace/plugins/myplugin/myplugin`

Remember to rebuild and reinstall your plugin each time you make changes to its code.
