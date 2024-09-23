package gsplug

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	pb "github.com/ssotops/gitspace-plugin-sdk/proto"
	"google.golang.org/protobuf/proto"
)

type PluginHandler interface {
	GetPluginInfo(req *pb.PluginInfoRequest) (*pb.PluginInfo, error)
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
