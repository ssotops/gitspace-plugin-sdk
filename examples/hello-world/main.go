package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ssotops/gitspace-plugin-sdk/gsplug"
	"github.com/ssotops/gitspace-plugin-sdk/logger"
	pb "github.com/ssotops/gitspace-plugin-sdk/proto"
	"google.golang.org/protobuf/proto"
)

type HelloWorldPlugin struct {
	logger *logger.RateLimitedLogger
}

func (p *HelloWorldPlugin) GetPluginInfo(req *pb.PluginInfoRequest) (*pb.PluginInfo, error) {
	p.logger.Info("GetPluginInfo called")
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
	pluginLogger, err := logger.NewRateLimitedLogger("hello-world")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	pluginLogger.Info("Hello World plugin starting")

	plugin := &HelloWorldPlugin{
		logger: pluginLogger,
	}

	for {
		pluginLogger.Debug("Waiting for message")
		msgType, msg, err := gsplug.ReadMessage(os.Stdin)
		if err != nil {
			if err == io.EOF {
				pluginLogger.Info("Received EOF, exiting")
				return
			}
			pluginLogger.Error("Error reading message", "error", err)
			continue
		}
		pluginLogger.Debug("Received message", "type", msgType, "content", fmt.Sprintf("%+v", msg))

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
			pluginLogger.Error("Error handling message", "error", err)
			continue
		}

		pluginLogger.Debug("Sending response", "type", msgType, "content", fmt.Sprintf("%+v", response))
		err = gsplug.WriteMessage(os.Stdout, response)
		if err != nil {
			pluginLogger.Error("Error writing response", "error", err)
		} else {
			pluginLogger.Debug("Response sent successfully")
		}

		// Flush stdout to ensure the message is sent immediately
		os.Stdout.Sync()
	}
}
