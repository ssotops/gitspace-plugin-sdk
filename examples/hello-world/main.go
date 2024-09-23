package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/log"
	"github.com/ssotops/gitspace-plugin-sdk/gsplug"
	pb "github.com/ssotops/gitspace-plugin-sdk/proto"
	"google.golang.org/protobuf/proto"
)

type HelloWorldPlugin struct{}

type MenuOption struct {
    Label   string `json:"label"`
    Command string `json:"command"`
}


func (p *HelloWorldPlugin) GetPluginInfo(req *pb.PluginInfoRequest) (*pb.PluginInfo, error) {
	log.Info("GetPluginInfo called")
	return &pb.PluginInfo{
		Name:    "Hello World Plugin",
		Version: "1.0.0",
	}, nil
}

func (p *HelloWorldPlugin) ExecuteCommand(req *pb.CommandRequest) (*pb.CommandResponse, error) {
	log.Info("ExecuteCommand called", "command", req.Command)
	switch req.Command {
	case "greet":
		return &pb.CommandResponse{
			Success: true,
			Result:  fmt.Sprintf("Hello, %s!", req.Parameters["name"]),
		}, nil
	case "customize":
		return &pb.CommandResponse{
			Success: true,
			Result:  fmt.Sprintf("%s, %s!", req.Parameters["greeting"], req.Parameters["name"]),
		}, nil
	default:
		return &pb.CommandResponse{
			Success:      false,
			ErrorMessage: "Unknown command",
		}, nil
	}
}

func (p *HelloWorldPlugin) GetMenu(req *pb.MenuRequest) (*pb.MenuResponse, error) {
	log.Info("GetMenu called")

	menuOptions := []MenuOption{
		{Label: "Simple Greeting", Command: "greet"},
		{Label: "Custom Greeting", Command: "customize"},
	}

	// Serialize the menu options to JSON
	menuBytes, err := json.Marshal(menuOptions)
	if err != nil {
		log.Error("Failed to marshal menu", "error", err)
		return nil, fmt.Errorf("failed to marshal menu: %w", err)
	}

	log.Info("Menu marshalled successfully", "size", len(menuBytes))

	return &pb.MenuResponse{
		MenuData: menuBytes,
	}, nil
}

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           log.DebugLevel,
	})
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
		}
		logger.Debug("Response sent")

		// Flush stdout to ensure the message is sent immediately
		os.Stdout.Sync()
	}
}
