// gsplug/sdk.go

package gsplug

import (
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sync"

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
    globalHandler     PluginHandler
    currentRequest    *pb.CommandRequest
    currentMenuOptions []MenuOption
    menuContextMutex  sync.RWMutex
    menuContextStack  []*MenuContext
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
        ParentMenu:       ctx.ParentMenu,
        AvailableCommands: menuItems,
    }
}

// ReadMessage reads a message from the given reader
func ReadMessage(r io.Reader) (uint32, proto.Message, error) {
    var msgType [1]byte
    _, err := r.Read(msgType[:])
    if err != nil {
        return 0, nil, fmt.Errorf("failed to read message type: %w", err)
    }
    log.Debug("Read message type", "type", msgType[0])

    var msgLen uint32
    err = binary.Read(r, binary.LittleEndian, &msgLen)
    if err != nil {
        return 0, nil, fmt.Errorf("failed to read message length: %w", err)
    }
    log.Debug("Read message length", "length", msgLen)

    data := make([]byte, msgLen)
    _, err = io.ReadFull(r, data)
    if err != nil {
        return 0, nil, fmt.Errorf("failed to read message data: %w", err)
    }
    log.Debug("Read message data", "dataLength", len(data))

    var msg proto.Message
    switch msgType[0] {
    case 1:
        msg = &pb.PluginInfoRequest{}
    case 2:
        msg = &pb.CommandRequest{}
    case 3:
        msg = &pb.MenuRequest{}
    default:
        return 0, nil, fmt.Errorf("unknown message type: %d", msgType[0])
    }

    err = proto.Unmarshal(data, msg)
    if err != nil {
        return 0, nil, fmt.Errorf("failed to unmarshal message: %w", err)
    }

    // Store current request if it's a command request
    if cmdReq, ok := msg.(*pb.CommandRequest); ok {
        currentRequest = cmdReq
    }

    return uint32(msgType[0]), msg, nil
}

// WriteMessage writes a message to the given writer
func WriteMessage(w io.Writer, msg proto.Message) error {
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

    log.Debug("Writing message type", "type", msgType)
    if _, err := w.Write([]byte{msgType}); err != nil {
        return fmt.Errorf("failed to write message type: %w", err)
    }

    log.Debug("Writing message length", "length", len(data))
    if err := binary.Write(w, binary.LittleEndian, uint32(len(data))); err != nil {
        return fmt.Errorf("failed to write message length: %w", err)
    }

    log.Debug("Writing message data", "dataLength", len(data))
    if _, err := w.Write(data); err != nil {
        return fmt.Errorf("failed to write message data: %w", err)
    }

    return nil
}

// RunPlugin runs the plugin main loop
func RunPlugin(handler PluginHandler) {
    SetPluginHandler(handler)

    for {
        msgType, msg, err := ReadMessage(os.Stdin)
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
            log.Error("Handler error", "error", handlerErr)
            continue
        }

        if err := WriteMessage(os.Stdout, response); err != nil {
            log.Error("Error writing response", "error", err)
            continue
        }

        // Ensure the response is sent immediately
        os.Stdout.Sync()
    }
}

// GetPluginLogDir returns the plugin's log directory path
func GetPluginLogDir(pluginName string) (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("failed to get user home directory: %w", err)
    }
    return filepath.Join(homeDir, ".ssot", "gitspace", "logs", pluginName), nil
}
