package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/ssotops/gitspace-plugin-sdk/gsplug"
	pb "github.com/ssotops/gitspace-plugin-sdk/proto"
)

type HelloWorldPlugin struct{}

func (p *HelloWorldPlugin) GetPluginInfo(req *pb.PluginInfoRequest) (*pb.PluginInfo, error) {
	log.Println("GetPluginInfo called")
	return &pb.PluginInfo{
		Name:    "Hello World Plugin",
		Version: "1.0.0",
	}, nil
}

func (p *HelloWorldPlugin) ExecuteCommand(req *pb.CommandRequest) (*pb.CommandResponse, error) {
	log.Printf("ExecuteCommand called with command: %s", req.Command)
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
	log.Println("GetMenu called")
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
		log.Printf("Form error: %v", err)
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
	log.SetOutput(os.Stderr)
	log.Println("Hello World plugin starting")

	// Create a channel to signal when to exit
	done := make(chan bool)

	// Run the plugin in a goroutine
	go func() {
		gsplug.RunPlugin(&HelloWorldPlugin{})
		done <- true
	}()

	// Wait for either input or timeout
	select {
	case <-done:
		log.Println("Plugin exited")
	case <-time.After(5 * time.Second):
		log.Println("No input received after 5 seconds. Exiting.")
	}
}
