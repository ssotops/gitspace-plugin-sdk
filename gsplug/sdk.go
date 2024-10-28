// gsplug/sdk.go

package gsplug

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	pb "github.com/ssotops/gitspace-plugin-sdk/proto"
	"google.golang.org/protobuf/proto"
)

// MenuContext tracks the current state of menu navigation
type MenuContext struct {
	CurrentMenu string
	ParentMenu  string
	Path        []string
	Options     []MenuOption
}

// MenuOption represents a menu item with potential submenus
type MenuOption struct {
	Label      string          `json:"label"`
	Command    string          `json:"command"`
	Parameters []ParameterInfo `json:"parameters,omitempty"`
	SubMenu    []MenuOption    `json:"sub_menu,omitempty"`
}

// ParameterInfo describes a command parameter
type ParameterInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// PluginHandler interface defines the required methods for a plugin
type PluginHandler interface {
	GetPluginInfo(*pb.PluginInfoRequest) (*pb.PluginInfo, error)
	ExecuteCommand(*pb.CommandRequest) (*pb.CommandResponse, error)
	GetMenu(*pb.MenuRequest) (*pb.MenuResponse, error)
	GetMenuContext(*pb.CommandRequest) (*MenuContext, error)
}

// Global state management
var (
	globalHandler      PluginHandler
	currentRequest     *pb.CommandRequest
	currentMenuOptions []MenuOption
	menuContextMutex   sync.RWMutex
	menuContextStack   []*MenuContext
)

// SetPluginHandler sets the global plugin handler
func SetPluginHandler(handler PluginHandler) {
	globalHandler = handler
}

// PushMenuContext adds a new menu context to the stack
func PushMenuContext(ctx *MenuContext) {
	menuContextMutex.Lock()
	defer menuContextMutex.Unlock()
	menuContextStack = append(menuContextStack, ctx)
}

// PopMenuContext removes and returns the top menu context
func PopMenuContext() *MenuContext {
	menuContextMutex.Lock()
	defer menuContextMutex.Unlock()
	if len(menuContextStack) == 0 {
		return nil
	}
	lastIdx := len(menuContextStack) - 1
	ctx := menuContextStack[lastIdx]
	menuContextStack = menuContextStack[:lastIdx]
	return ctx
}

// GetCurrentMenuContext returns the current menu context
func GetCurrentMenuContext() *MenuContext {
	menuContextMutex.RLock()
	defer menuContextMutex.RUnlock()
	if len(menuContextStack) == 0 {
		return nil
	}
	return menuContextStack[len(menuContextStack)-1]
}

// BuildNavigationContext creates a NavigationContext from MenuContext
func BuildNavigationContext(ctx *MenuContext) *pb.NavigationContext {
	if ctx == nil {
		return nil
	}

	menuItems := make([]*pb.MenuItem, 0, len(ctx.Options))
	for _, opt := range ctx.Options {
		item := &pb.MenuItem{
			Label:   opt.Label,
			Command: opt.Command,
		}

		if len(opt.SubMenu) > 0 {
			item.SubmenuId = fmt.Sprintf("%s_%s", ctx.CurrentMenu, opt.Command)
		}

		if len(opt.Parameters) > 0 {
			params := make([]*pb.ParameterInfo, 0, len(opt.Parameters))
			for _, p := range opt.Parameters {
				params = append(params, &pb.ParameterInfo{
					Name:        p.Name,
					Description: p.Description,
					Required:    p.Required,
				})
			}
			item.Parameters = params
		}

		menuItems = append(menuItems, item)
	}

	return &pb.NavigationContext{
		CurrentMenu:       ctx.CurrentMenu,
		ParentMenu:        ctx.ParentMenu,
		AvailableCommands: menuItems,
	}
}

// plugin_sdk/gsplug/sdk.go

func RunPlugin(handler PluginHandler) {
	SetPluginHandler(handler)

	// Create buffered reader/writer with detailed logging and metrics
	reader := bufio.NewReaderSize(os.Stdin, 1024*1024)  // 1MB buffer
	writer := bufio.NewWriterSize(os.Stdout, 1024*1024) // 1MB buffer

	log.Debug("Plugin IO initialization",
		"readerBufferSize", reader.Size(),
		"writerBufferSize", writer.Size(),
		"stdin", fmt.Sprintf("%T", os.Stdin),
		"stdout", fmt.Sprintf("%T", os.Stdout))

	// Channel to track if we should exit
	done := make(chan struct{})
	log.Debug("Created exit channel")

	// Handle signals gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.Debug("Signal handlers initialized",
		"signals", []os.Signal{os.Interrupt, syscall.SIGTERM})

	go func() {
		sig := <-sigChan
		log.Info("Received signal, initiating shutdown",
			"signal", sig,
			"signalType", fmt.Sprintf("%T", sig))
		close(done)
	}()

	// Use WaitGroup to ensure clean shutdown
	var wg sync.WaitGroup
	wg.Add(1)

	// Message processing metrics
	var messageCount uint64
	processStart := time.Now()

	go func() {
		defer func() {
			log.Debug("Plugin message loop ending",
				"totalMessages", messageCount,
				"uptime", time.Since(processStart))
			wg.Done()
		}()

		for {
			select {
			case <-done:
				log.Info("Received shutdown signal, stopping message loop")
				return
			default:
				startTime := time.Now()
				log.Debug("Starting message read cycle",
					"messageCount", messageCount,
					"readerBuffered", reader.Buffered())

				msgType, msg, err := ReadMessage(reader)
				if err != nil {
					if err == io.EOF {
						log.Info("Received EOF, exiting normally",
							"totalMessages", messageCount,
							"uptime", time.Since(processStart))
						return
					}
					log.Error("Error reading message",
						"error", err,
						"errorType", fmt.Sprintf("%T", err),
						"messageCount", messageCount)
					continue
				}

				log.Debug("Message received",
					"messageType", msgType,
					"messageSize", proto.Size(msg),
					"messageTypeName", fmt.Sprintf("%T", msg))

				var response proto.Message
				var handlerErr error

				handlerStart := time.Now()
				switch msgType {
				case 1:
					log.Debug("Handling GetPluginInfo request")
					response, handlerErr = handler.GetPluginInfo(msg.(*pb.PluginInfoRequest))
				case 2:
					log.Debug("Handling ExecuteCommand request")
					response, handlerErr = handler.ExecuteCommand(msg.(*pb.CommandRequest))
				case 3:
					log.Debug("Handling GetMenu request")
					response, handlerErr = handler.GetMenu(msg.(*pb.MenuRequest))
				default:
					log.Error("Unknown message type",
						"type", msgType,
						"rawMessage", fmt.Sprintf("%+v", msg))
					continue
				}

				handlerDuration := time.Since(handlerStart)
				log.Debug("Handler execution completed",
					"duration", handlerDuration,
					"error", handlerErr != nil)

				if handlerErr != nil {
					log.Error("Handler error",
						"error", handlerErr,
						"errorType", fmt.Sprintf("%T", handlerErr),
						"messageType", msgType)
					// Create error response instead of continuing
					switch msgType {
					case 1:
						response = &pb.PluginInfo{
							Name:    "Error",
							Version: handlerErr.Error(),
						}
					case 2:
						response = &pb.CommandResponse{
							Success:      false,
							ErrorMessage: handlerErr.Error(),
						}
					case 3:
						errorJSON := fmt.Sprintf(`{"error":"%s"}`, handlerErr.Error())
						response = &pb.MenuResponse{
							MenuData: []byte(errorJSON),
						}
					}
				}

				// Use mutex to synchronize writes
				writeMutex.Lock()
				writeStart := time.Now()
				if err := WriteMessage(writer, response); err != nil {
					log.Error("Error writing response",
						"error", err,
						"errorType", fmt.Sprintf("%T", err),
						"responseType", fmt.Sprintf("%T", response))
				} else {
					log.Debug("Response written successfully",
						"duration", time.Since(writeStart),
						"responseSize", proto.Size(response))
				}
				writeMutex.Unlock()

				messageCount++
				cycleDuration := time.Since(startTime)
				log.Debug("Message cycle completed",
					"duration", cycleDuration,
					"messageNumber", messageCount)
			}
		}
	}()

	log.Info("Plugin message loop started",
		"pid", os.Getpid(),
		"ppid", os.Getppid())

	wg.Wait()
	log.Info("Plugin shutdown complete",
		"totalMessages", messageCount,
		"uptime", time.Since(processStart))
}

var writeMutex sync.Mutex

// WriteMessage writes a message to the given writer
func WriteMessage(w io.Writer, msg proto.Message) error {
	writeStart := time.Now()
	log.Debug("Starting WriteMessage operation",
		"messageType", fmt.Sprintf("%T", msg))

	// Add navigation context to command responses
	if resp, ok := msg.(*pb.CommandResponse); ok {
		if globalHandler != nil && currentRequest != nil {
			navigationStart := time.Now()
			if ctx, err := globalHandler.GetMenuContext(currentRequest); err == nil {
				resp.Navigation = BuildNavigationContext(ctx)
				log.Debug("Added navigation context",
					"duration", time.Since(navigationStart))
			} else {
				log.Warn("Failed to get menu context",
					"error", err,
					"duration", time.Since(navigationStart))
			}
		}
	}

	// Marshal message
	marshalStart := time.Now()
	data, err := proto.Marshal(msg)
	if err != nil {
		log.Error("Failed to marshal message",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"messageType", fmt.Sprintf("%T", msg),
			"duration", time.Since(marshalStart))
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	log.Debug("Marshaled message successfully",
		"dataLength", len(data),
		"duration", time.Since(marshalStart))

	msgType := uint8(0)
	switch msg.(type) {
	case *pb.PluginInfo:
		msgType = 1
	case *pb.CommandResponse:
		msgType = 2
	case *pb.MenuResponse:
		msgType = 3
	default:
		log.Error("Unknown message type for writing",
			"messageType", fmt.Sprintf("%T", msg))
		return fmt.Errorf("unknown message type: %T", msg)
	}

	// Create buffer for complete message
	bufferStart := time.Now()
	buf := new(bytes.Buffer)

	// Write message type
	if err := buf.WriteByte(msgType); err != nil {
		log.Error("Failed to write message type",
			"error", err,
			"type", msgType,
			"duration", time.Since(bufferStart))
		return fmt.Errorf("failed to write message type: %w", err)
	}

	// Write message length
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(data))); err != nil {
		log.Error("Failed to write message length",
			"error", err,
			"length", len(data),
			"duration", time.Since(bufferStart))
		return fmt.Errorf("failed to write message length: %w", err)
	}

	// Write message data
	if _, err := buf.Write(data); err != nil {
		log.Error("Failed to write message data",
			"error", err,
			"dataLength", len(data),
			"duration", time.Since(bufferStart))
		return fmt.Errorf("failed to write message data: %w", err)
	}
	log.Debug("Buffer preparation complete",
		"bufferSize", buf.Len(),
		"duration", time.Since(bufferStart))

	// Write the entire buffer at once
	writeBufferStart := time.Now()
	if _, err := io.Copy(w, buf); err != nil {
		log.Error("Failed to write buffer to output",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"bufferSize", buf.Len(),
			"duration", time.Since(writeBufferStart))
		return fmt.Errorf("failed to write message: %w", err)
	}

	// If it's a buffered writer, flush it
	if bw, ok := w.(*bufio.Writer); ok {
		flushStart := time.Now()
		if err := bw.Flush(); err != nil {
			log.Error("Failed to flush writer",
				"error", err,
				"errorType", fmt.Sprintf("%T", err),
				"duration", time.Since(flushStart))
			return fmt.Errorf("failed to flush writer: %w", err)
		}
		log.Debug("Writer flushed successfully",
			"duration", time.Since(flushStart))
	}

	totalDuration := time.Since(writeStart)
	log.Debug("Write operation completed successfully",
		"messageType", fmt.Sprintf("%T", msg),
		"dataLength", len(data),
		"totalDuration", totalDuration)

	return nil
}

func ReadMessage(r *bufio.Reader) (uint32, proto.Message, error) {
	if r == nil {
		log.Error("Reader cannot be nil")
		return 0, nil, fmt.Errorf("reader cannot be nil")
	}

	log.Debug("Starting ReadMessage operation",
		"readerBufferSize", r.Size(),
		"bufferedBytes", r.Buffered(),
		"readerType", fmt.Sprintf("%T", r))

	readStart := time.Now()

	// Read message type
	msgType, err := r.ReadByte()
	if err != nil {
		if err == io.EOF {
			log.Debug("Received EOF while reading message type",
				"duration", time.Since(readStart))
			return 0, nil, err
		}
		log.Error("Failed to read message type",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"duration", time.Since(readStart))
		return 0, nil, fmt.Errorf("failed to read message type: %w", err)
	}
	log.Debug("Read message type successfully",
		"type", msgType,
		"typeByte", fmt.Sprintf("%x", msgType),
		"duration", time.Since(readStart))

	// Read message length
	var msgLen uint32
	if err := binary.Read(r, binary.LittleEndian, &msgLen); err != nil {
		log.Error("Failed to read message length",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"duration", time.Since(readStart))
		return 0, nil, fmt.Errorf("failed to read message length: %w", err)
	}
	log.Debug("Read message length successfully",
		"length", msgLen,
		"duration", time.Since(readStart))

	// Validate message length
	maxMessageSize := uint32(10 * 1024 * 1024) // 10MB limit
	if msgLen > maxMessageSize {
		log.Error("Message length exceeds limit",
			"length", msgLen,
			"limit", maxMessageSize,
			"duration", time.Since(readStart))
		return 0, nil, fmt.Errorf("message too large: %d bytes (limit: %d bytes)", msgLen, maxMessageSize)
	}

	// Read message data
	data := make([]byte, msgLen)
	n, err := io.ReadFull(r, data)
	if err != nil {
		log.Error("Failed to read message data",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"bytesRead", n,
			"expectedLength", msgLen,
			"duration", time.Since(readStart))
		return 0, nil, fmt.Errorf("failed to read message data: %w", err)
	}
	log.Debug("Read message data successfully",
		"bytesRead", n,
		"data", fmt.Sprintf("%x", data),
		"duration", time.Since(readStart))

	// Create appropriate message type
	var msg proto.Message
	switch msgType {
	case 1:
		msg = &pb.PluginInfoRequest{}
		log.Debug("Created PluginInfoRequest message")
	case 2:
		msg = &pb.CommandRequest{}
		log.Debug("Created CommandRequest message")
	case 3:
		msg = &pb.MenuRequest{}
		log.Debug("Created MenuRequest message")
	default:
		log.Error("Unknown message type",
			"type", msgType,
			"duration", time.Since(readStart))
		return 0, nil, fmt.Errorf("unknown message type: %d", msgType)
	}

	// Unmarshal the message
	unmarshalStart := time.Now()
	if err := proto.Unmarshal(data, msg); err != nil {
		log.Error("Failed to unmarshal message",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"messageType", fmt.Sprintf("%T", msg),
			"duration", time.Since(unmarshalStart))
		return 0, nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	totalDuration := time.Since(readStart)
	log.Debug("Message read and unmarshaled successfully",
		"messageType", fmt.Sprintf("%T", msg),
		"readDuration", totalDuration,
		"unmarshalDuration", time.Since(unmarshalStart),
		"messageSize", len(data))

	return uint32(msgType), msg, nil
}

// GetPluginLogDir returns the plugin's log directory path
func GetPluginLogDir(pluginName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ssot", "gitspace", "logs", pluginName), nil
}
