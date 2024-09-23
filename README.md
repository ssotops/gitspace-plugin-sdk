# Gitspace Plugin SDK

The Gitspace Plugin SDK is a toolkit for developing plugins for the Gitspace application. It provides a set of tools and interfaces that allow developers to create custom plugins that can be seamlessly integrated into Gitspace.

## Table of Contents

1. [Installation](#installation)
2. [Usage](#usage)
3. [Plugin Structure](#plugin-structure)
4. [API Reference](#api-reference)
5. [Example Plugin](#example-plugin)
6. [Contributing](#contributing)
7. [License](#license)

## Installation

To use the Gitspace Plugin SDK in your project, run:

```bash
go get github.com/ssotops/gitspace-plugin-sdk
```

## Usage
To create a new plugin, you'll need to implement the PluginHandler interface provided by the SDK:

```go
type PluginHandler interface {
    GetPluginInfo(*pb.PluginInfoRequest) (*pb.PluginInfo, error)
    ExecuteCommand(*pb.CommandRequest) (*pb.CommandResponse, error)
    GetMenu(*pb.MenuRequest) (*pb.MenuResponse, error)
}
```

## Plugin Structure
A typical plugin structure looks like this:
```
my-plugin/
├── main.go
└── gitspace-plugin.toml
```

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

##API Reference
- **GetPluginInfo**
> This method should return information about your plugin, including its name and version.
- **ExecuteCommand**
> This method is called when Gitspace wants to execute a command provided by your plugin.
- **GetMenu**
> This method should return a menu structure that Gitspace will display to the user.

## Example Plugin
Here's a simple example of a plugin implementation:

```go
package main

import (
    "encoding/json"
    "fmt"

    "github.com/ssotops/gitspace-plugin-sdk/gsplug"
    pb "github.com/ssotops/gitspace-plugin-sdk/proto"
)

type MyPlugin struct{}

func (p *MyPlugin) GetPluginInfo(req *pb.PluginInfoRequest) (*pb.PluginInfo, error) {
    return &pb.PluginInfo{
        Name:    "My Plugin",
        Version: "1.0.0",
    }, nil
}

func (p *MyPlugin) ExecuteCommand(req *pb.CommandRequest) (*pb.CommandResponse, error) {
    switch req.Command {
    case "hello":
        return &pb.CommandResponse{
            Success: true,
            Result:  "Hello from My Plugin!",
        }, nil
    default:
        return &pb.CommandResponse{
            Success:      false,
            ErrorMessage: "Unknown command",
        }, nil
    }
}

func (p *MyPlugin) GetMenu(req *pb.MenuRequest) (*pb.MenuResponse, error) {
    menu := []map[string]string{
        {"label": "Say Hello", "command": "hello"},
    }
    menuBytes, err := json.Marshal(menu)
    if err != nil {
        return nil, err
    }
    return &pb.MenuResponse{
        MenuData: menuBytes,
    }, nil
}

func main() {
    plugin := &MyPlugin{}
    gsplug.RunPlugin(plugin)
}
```
