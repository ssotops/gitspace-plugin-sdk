package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/ssotops/gitspace-plugin-sdk/gsplug"
	pb "github.com/ssotops/gitspace-plugin-sdk/proto"
)

type HelloWorldPlugin struct{}

func (p *HelloWorldPlugin) GetPluginInfo(req *pb.PluginInfoRequest) (*pb.PluginInfo, error) {
	return &pb.PluginInfo{
		Name:    "Hello World Plugin",
		Version: "1.0.0",
	}, nil
}

func (p *HelloWorldPlugin) ExecuteCommand(req *pb.CommandRequest) (*pb.CommandResponse, error) {
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
	var name, greeting string
	var uppercase bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter your name").
				Value(&name),
			huh.NewSelect[string]().
				Title("Choose a greeting").
				Options(
					huh.NewOption("Hello", "Hello"),
					huh.NewOption("Hi", "Hi"),
					huh.NewOption("Hey", "Hey"),
				).
				Value(&greeting),
			huh.NewConfirm().
				Title("Uppercase the greeting?").
				Value(&uppercase),
		),
	)

	err := form.Run()
	if err != nil {
		return &pb.MenuResponse{
			Items: []*pb.MenuItem{
				{Label: "Error", Command: "greet"},
			},
		}, nil
	}

	if uppercase {
		greeting = strings.ToUpper(greeting)
	}

	return &pb.MenuResponse{
		Items: []*pb.MenuItem{
			{Label: "Simple Greeting", Command: "greet"},
			{Label: "Custom Greeting", Command: "customize"},
		},
	}, nil
}

func main() {
	gsplug.RunPlugin(&HelloWorldPlugin{})
}
