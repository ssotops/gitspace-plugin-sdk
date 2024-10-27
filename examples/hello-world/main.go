// gitspace-plugin-sdk/examples/hello-world/main.go

package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/charmbracelet/log"
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
	p.logger.Debug("ExecuteCommand called", "command", req.Command, "params", req.Parameters)

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
			ErrorMessage: fmt.Sprintf("Unknown command: %s", req.Command),
		}, nil
	}
}

func (p *HelloWorldPlugin) GetMenu(req *pb.MenuRequest) (*pb.MenuResponse, error) {
	p.logger.Debug("GetMenu called")

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

	menuBytes, err := proto.Marshal(&pb.MenuResponse{
		MenuData: []byte(fmt.Sprintf("%+v", menuOptions)),
	})
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
	pluginLogger.SetLogLevel(log.DebugLevel)

	pluginLogger.Info("Hello World plugin starting up")

	dir, err := os.Getwd()
	if err != nil {
		dir = "unknown"
	}
	pluginLogger.Debug("Process information",
		"pid", os.Getpid(),
		"ppid", os.Getppid(),
		"uid", os.Getuid(),
		"gid", os.Getgid(),
		"dir", dir)

	plugin := &HelloWorldPlugin{
		logger: pluginLogger,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)

	pluginLogger.Debug("Checking IO streams",
		"stdin", fmt.Sprintf("%T", os.Stdin),
		"stdout", fmt.Sprintf("%T", os.Stdout),
		"stderr", fmt.Sprintf("%T", os.Stderr))

	go func() {
		reader := bufio.NewReader(os.Stdin)
		writer := bufio.NewWriter(os.Stdout)

		pluginLogger.Debug("Created IO buffers",
			"readerSize", reader.Size(),
			"writerSize", writer.Size())

		for {
			pluginLogger.Debug("Waiting to read message type")
			msgTypeByte := make([]byte, 1)
			n, err := io.ReadFull(reader, msgTypeByte)
			if err != nil {
				if err == io.EOF {
					pluginLogger.Info("Received EOF, exiting normally")
					errChan <- nil
					return
				}
				errChan <- fmt.Errorf("failed to read message type: %w", err)
				return
			}
			pluginLogger.Debug("Read message type byte",
				"bytesRead", n,
				"messageType", msgTypeByte[0])

			pluginLogger.Debug("Reading message length")
			var msgLen uint32
			if err := binary.Read(reader, binary.LittleEndian, &msgLen); err != nil {
				errChan <- fmt.Errorf("failed to read message length: %w", err)
				return
			}
			pluginLogger.Debug("Read message length", "length", msgLen)

			data := make([]byte, msgLen)
			n, err = io.ReadFull(reader, data)
			if err != nil {
				errChan <- fmt.Errorf("failed to read message data: %w", err)
				return
			}
			pluginLogger.Debug("Read message data",
				"bytesRead", n,
				"dataLength", len(data),
				"data", fmt.Sprintf("%x", data))

			var response proto.Message
			msgType := uint32(msgTypeByte[0])
			switch msgType {
			case 1:
				pluginLogger.Debug("Handling GetPluginInfo request")
				req := &pb.PluginInfoRequest{}
				if err := proto.Unmarshal(data, req); err != nil {
					errChan <- fmt.Errorf("failed to unmarshal GetPluginInfo request: %w", err)
					return
				}
				response, err = plugin.GetPluginInfo(req)

			case 2:
				pluginLogger.Debug("Handling ExecuteCommand request")
				req := &pb.CommandRequest{}
				if err := proto.Unmarshal(data, req); err != nil {
					errChan <- fmt.Errorf("failed to unmarshal ExecuteCommand request: %w", err)
					return
				}
				response, err = plugin.ExecuteCommand(req)

			case 3:
				pluginLogger.Debug("Handling GetMenu request")
				req := &pb.MenuRequest{}
				if err := proto.Unmarshal(data, req); err != nil {
					errChan <- fmt.Errorf("failed to unmarshal GetMenu request: %w", err)
					return
				}
				response, err = plugin.GetMenu(req)

			default:
				errChan <- fmt.Errorf("unknown message type: %d", msgType)
				return
			}

			if err != nil {
				errChan <- fmt.Errorf("error handling message type %d: %w", msgType, err)
				return
			}

			// Marshal and send response
			pluginLogger.Debug("Marshaling response",
				"type", fmt.Sprintf("%T", response))
			responseData, err := proto.Marshal(response)
			if err != nil {
				errChan <- fmt.Errorf("failed to marshal response: %w", err)
				return
			}

			var writeMutex sync.Mutex
			writeMutex.Lock()
			defer writeMutex.Unlock()

			pluginLogger.Debug("Writing response type", "type", msgType)
			if _, err := writer.Write([]byte{byte(msgType)}); err != nil {
				errChan <- fmt.Errorf("failed to write response type: %w", err)
				return
			}

			pluginLogger.Debug("Writing response length", "length", len(responseData))
			if err := binary.Write(writer, binary.LittleEndian, uint32(len(responseData))); err != nil {
				errChan <- fmt.Errorf("failed to write response length: %w", err)
				return
			}

			pluginLogger.Debug("Writing response data",
				"dataLength", len(responseData),
				"data", fmt.Sprintf("%x", responseData))
			if _, err := writer.Write(responseData); err != nil {
				errChan <- fmt.Errorf("failed to write response data: %w", err)
				return
			}

			pluginLogger.Debug("Flushing writer")
			if err := writer.Flush(); err != nil {
				errChan <- fmt.Errorf("failed to flush writer: %w", err)
				return
			}
			pluginLogger.Debug("Response sent successfully")
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			pluginLogger.Error("Plugin error",
				"error", err,
				"errorType", fmt.Sprintf("%T", err))
			os.Exit(1)
		}
		pluginLogger.Info("Plugin exiting normally")
	case sig := <-sigChan:
		pluginLogger.Info("Received signal, shutting down",
			"signal", sig,
			"signalType", fmt.Sprintf("%T", sig))
	}
}
