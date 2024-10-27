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

// gitspace-plugin-sdk/gsplug/sdk.go
func RunPlugin(handler PluginHandler) {
	SetPluginHandler(handler)

	// Create buffered reader for stdin
	reader := bufio.NewReaderSize(os.Stdin, 1024*1024) // 1MB buffer

	// Create buffered writer for stdout
	writer := bufio.NewWriterSize(os.Stdout, 1024*1024) // 1MB buffer

	// Channel to track if we should exit
	done := make(chan struct{})

	// Handle signals gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		close(done)
	}()

	// Use WaitGroup to ensure clean shutdown
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				// Change gsplug.ReadMessage to just ReadMessage since we're in the same package
				msgType, msg, err := ReadMessage(reader)
				if err != nil {
					if err == io.EOF {
						log.Info("Received EOF, exiting")
						return
					}
					log.Error("Error reading message", "error", err)
					continue
				}

				var response proto.Message
				var handlerErr error

				switch msgType {
				case 1:
					response, handlerErr = handler.GetPluginInfo(msg.(*pb.PluginInfoRequest))
				case 2:
					response, handlerErr = handler.ExecuteCommand(msg.(*pb.CommandRequest))
				case 3:
					response, handlerErr = handler.GetMenu(msg.(*pb.MenuRequest))
				default:
					log.Error("Unknown message type", "type", msgType)
					continue
				}

				if handlerErr != nil {
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
				if err := WriteMessage(writer, response); err != nil {
					log.Error("Error writing response", "error", err)
				}
				writeMutex.Unlock()
			}
		}
	}()

	wg.Wait()
}

var writeMutex sync.Mutex

// WriteMessage writes a message to the given writer
func WriteMessage(w io.Writer, msg proto.Message) error {
	// Create a buffered writer if it's not already one
	var bw *bufio.Writer
	if bufWriter, ok := w.(*bufio.Writer); ok {
		bw = bufWriter
	} else {
		bw = bufio.NewWriterSize(w, 1024*1024) // 1MB buffer
	}

	// Add navigation context to command responses
	if resp, ok := msg.(*pb.CommandResponse); ok {
		if globalHandler != nil && currentRequest != nil {
			if ctx, err := globalHandler.GetMenuContext(currentRequest); err == nil {
				resp.Navigation = BuildNavigationContext(ctx)
			}
		}
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	log.Debug("Marshaled message", "dataLength", len(data))

	msgType := uint8(0)
	switch msg.(type) {
	case *pb.PluginInfo:
		msgType = 1
	case *pb.CommandResponse:
		msgType = 2
	case *pb.MenuResponse:
		msgType = 3
	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}

	// Write everything in one buffer
	buf := new(bytes.Buffer)

	// Write message type
	if err := buf.WriteByte(msgType); err != nil {
		return fmt.Errorf("failed to write message type: %w", err)
	}

	// Write message length
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(data))); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}

	// Write message data
	if _, err := buf.Write(data); err != nil {
		return fmt.Errorf("failed to write message data: %w", err)
	}

	// Write the entire buffer at once and flush
	if _, err := io.Copy(bw, buf); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := bw.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	return nil
}

// gitspace-plugin-sdk/gsplug/sdk.go

func ReadMessage(r *bufio.Reader) (uint32, proto.Message, error) {
	if r == nil {
		return 0, nil, fmt.Errorf("reader cannot be nil")
	}

	log.Debug("Starting ReadMessage operation",
		"readerBufferSize", r.Size(),
		"readerType", fmt.Sprintf("%T", r))

	// Read message type
	msgType, err := r.ReadByte()
	if err != nil {
		if err == io.EOF {
			log.Debug("Received EOF while reading message type")
			return 0, nil, err
		}
		log.Error("Failed to read message type",
			"error", err,
			"errorType", fmt.Sprintf("%T", err))
		return 0, nil, fmt.Errorf("failed to read message type: %w", err)
	}
	log.Debug("Read message type successfully",
		"type", msgType,
		"typeByte", fmt.Sprintf("%x", msgType))

	// Read message length
	var msgLen uint32
	if err := binary.Read(r, binary.LittleEndian, &msgLen); err != nil {
		log.Error("Failed to read message length",
			"error", err,
			"errorType", fmt.Sprintf("%T", err))
		return 0, nil, fmt.Errorf("failed to read message length: %w", err)
	}
	log.Debug("Read message length successfully", "length", msgLen)

	// Validate message length
	if msgLen > 10*1024*1024 { // 10MB limit
		log.Error("Message length exceeds limit",
			"length", msgLen,
			"limit", 10*1024*1024)
		return 0, nil, fmt.Errorf("message too large: %d bytes", msgLen)
	}

	// Read message data
	data := make([]byte, msgLen)
	n, err := io.ReadFull(r, data)
	if err != nil {
		log.Error("Failed to read message data",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"bytesRead", n,
			"expectedLength", msgLen)
		return 0, nil, fmt.Errorf("failed to read message data: %w", err)
	}
	log.Debug("Read message data successfully",
		"bytesRead", n,
		"data", fmt.Sprintf("%x", data))

	// Create appropriate message type
	var msg proto.Message
	switch msgType {
	case 1:
		msg = &pb.PluginInfoRequest{}
	case 2:
		msg = &pb.CommandRequest{}
	case 3:
		msg = &pb.MenuRequest{}
	default:
		log.Error("Unknown message type",
			"type", msgType)
		return 0, nil, fmt.Errorf("unknown message type: %d", msgType)
	}

	// Unmarshal the message
	if err := proto.Unmarshal(data, msg); err != nil {
		log.Error("Failed to unmarshal message",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"messageType", fmt.Sprintf("%T", msg))
		return 0, nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	log.Debug("Message unmarshaled successfully",
		"messageType", fmt.Sprintf("%T", msg))

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
