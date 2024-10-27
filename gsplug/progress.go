// gitspace-plugin-sdk/gsplug/progress.go

package gsplug

import (
    "time"
    pb "github.com/ssotops/gitspace-plugin-sdk/proto"
)

type ProgressEmitter struct {
    sendResponse func(*pb.CommandResponse) error
}

func NewProgressEmitter(sendFn func(*pb.CommandResponse) error) *ProgressEmitter {
    return &ProgressEmitter{sendResponse: sendFn}
}

func (p *ProgressEmitter) Emit(phase, step, status, message string) error {
    return p.sendResponse(&pb.CommandResponse{
        Success: true,  // Keep connection alive
        Progress: &pb.ProgressUpdate{
            Phase:     phase,
            Step:      step,
            Status:    status,
            Message:   message,
            Timestamp: time.Now().Format(time.RFC3339),
        },
    })
}
