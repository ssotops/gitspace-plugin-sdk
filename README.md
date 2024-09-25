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
[hello-world plugin](https://github.com/ssotops/gitspace-plugin-sdk/blob/master/examples/hello-world/main.go)


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
