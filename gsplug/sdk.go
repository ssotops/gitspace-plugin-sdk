package gsplug

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	pb "github.com/ssotops/gitspace-plugin-sdk/proto"
	"google.golang.org/protobuf/proto"
)

type PluginHandler interface {
	GetPluginInfo(*pb.PluginInfoRequest) (*pb.PluginInfo, error)
	ExecuteCommand(*pb.CommandRequest) (*pb.CommandResponse, error)
	GetMenu(*pb.MenuRequest) (*pb.MenuResponse, error)
}

type MenuOption struct {
	Label      string          `json:"label"`
	Command    string          `json:"command"`
	Parameters []ParameterInfo `json:"parameters,omitempty"`
	SubMenu    []MenuOption    `json:"sub_menu,omitempty"`
}

type ParameterInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

func RunPlugin(handler PluginHandler) {
	for {
		msgType, msg, err := readMessage(os.Stdin)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading message: %v\n", err)
			continue
		}

		var response proto.Message
		var handlerErr error

		switch msgType {
		case 1:
			req := msg.(*pb.PluginInfoRequest)
			response, handlerErr = handler.GetPluginInfo(req)
		default:
			fmt.Fprintf(os.Stderr, "Unknown message type: %d\n", msgType)
			continue
		}

		if handlerErr != nil {
			fmt.Fprintf(os.Stderr, "Handler error: %v\n", handlerErr)
			continue
		}

		err = writeMessage(os.Stdout, response)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing response: %v\n", err)
		}
	}
}

func readMessage(r io.Reader) (uint32, proto.Message, error) {
	var msgType uint32
	err := binary.Read(r, binary.LittleEndian, &msgType)
	if err != nil {
		return 0, nil, err
	}

	var msgLen uint32
	err = binary.Read(r, binary.LittleEndian, &msgLen)
	if err != nil {
		return 0, nil, err
	}

	msgData := make([]byte, msgLen)
	_, err = io.ReadFull(r, msgData)
	if err != nil {
		return 0, nil, err
	}

	var msg proto.Message
	switch msgType {
	case 1:
		msg = &pb.PluginInfoRequest{}
	default:
		return 0, nil, fmt.Errorf("unknown message type: %d", msgType)
	}

	err = proto.Unmarshal(msgData, msg)
	if err != nil {
		return 0, nil, err
	}

	return msgType, msg, nil
}

func writeMessage(w io.Writer, msg proto.Message) error {
	msgData, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.LittleEndian, uint32(len(msgData)))
	if err != nil {
		return err
	}

	_, err = w.Write(msgData)
	return err
}

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
	log.Debug("Read message data", "dataLength", len(data), "rawData", fmt.Sprintf("%x", data))

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

	return uint32(msgType[0]), msg, nil
}

func WriteMessage(w io.Writer, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	log.Debug("Marshaled message", "dataLength", len(data), "rawData", fmt.Sprintf("%x", data))

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

func GetPluginLogDir(pluginName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ssot", "gitspace", "logs", pluginName), nil
}
