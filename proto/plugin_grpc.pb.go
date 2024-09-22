// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.28.1
// source: proto/plugin.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	PluginService_GetPluginInfo_FullMethodName  = "/gitspace.plugin.PluginService/GetPluginInfo"
	PluginService_ExecuteCommand_FullMethodName = "/gitspace.plugin.PluginService/ExecuteCommand"
	PluginService_GetMenu_FullMethodName        = "/gitspace.plugin.PluginService/GetMenu"
)

// PluginServiceClient is the client API for PluginService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PluginServiceClient interface {
	GetPluginInfo(ctx context.Context, in *PluginInfoRequest, opts ...grpc.CallOption) (*PluginInfo, error)
	ExecuteCommand(ctx context.Context, in *CommandRequest, opts ...grpc.CallOption) (*CommandResponse, error)
	GetMenu(ctx context.Context, in *MenuRequest, opts ...grpc.CallOption) (*MenuResponse, error)
}

type pluginServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPluginServiceClient(cc grpc.ClientConnInterface) PluginServiceClient {
	return &pluginServiceClient{cc}
}

func (c *pluginServiceClient) GetPluginInfo(ctx context.Context, in *PluginInfoRequest, opts ...grpc.CallOption) (*PluginInfo, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PluginInfo)
	err := c.cc.Invoke(ctx, PluginService_GetPluginInfo_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) ExecuteCommand(ctx context.Context, in *CommandRequest, opts ...grpc.CallOption) (*CommandResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CommandResponse)
	err := c.cc.Invoke(ctx, PluginService_ExecuteCommand_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) GetMenu(ctx context.Context, in *MenuRequest, opts ...grpc.CallOption) (*MenuResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MenuResponse)
	err := c.cc.Invoke(ctx, PluginService_GetMenu_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PluginServiceServer is the server API for PluginService service.
// All implementations must embed UnimplementedPluginServiceServer
// for forward compatibility.
type PluginServiceServer interface {
	GetPluginInfo(context.Context, *PluginInfoRequest) (*PluginInfo, error)
	ExecuteCommand(context.Context, *CommandRequest) (*CommandResponse, error)
	GetMenu(context.Context, *MenuRequest) (*MenuResponse, error)
	mustEmbedUnimplementedPluginServiceServer()
}

// UnimplementedPluginServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedPluginServiceServer struct{}

func (UnimplementedPluginServiceServer) GetPluginInfo(context.Context, *PluginInfoRequest) (*PluginInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPluginInfo not implemented")
}
func (UnimplementedPluginServiceServer) ExecuteCommand(context.Context, *CommandRequest) (*CommandResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExecuteCommand not implemented")
}
func (UnimplementedPluginServiceServer) GetMenu(context.Context, *MenuRequest) (*MenuResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMenu not implemented")
}
func (UnimplementedPluginServiceServer) mustEmbedUnimplementedPluginServiceServer() {}
func (UnimplementedPluginServiceServer) testEmbeddedByValue()                       {}

// UnsafePluginServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PluginServiceServer will
// result in compilation errors.
type UnsafePluginServiceServer interface {
	mustEmbedUnimplementedPluginServiceServer()
}

func RegisterPluginServiceServer(s grpc.ServiceRegistrar, srv PluginServiceServer) {
	// If the following call pancis, it indicates UnimplementedPluginServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&PluginService_ServiceDesc, srv)
}

func _PluginService_GetPluginInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PluginInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).GetPluginInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_GetPluginInfo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).GetPluginInfo(ctx, req.(*PluginInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_ExecuteCommand_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommandRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ExecuteCommand(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ExecuteCommand_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ExecuteCommand(ctx, req.(*CommandRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_GetMenu_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MenuRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).GetMenu(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_GetMenu_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).GetMenu(ctx, req.(*MenuRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PluginService_ServiceDesc is the grpc.ServiceDesc for PluginService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PluginService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gitspace.plugin.PluginService",
	HandlerType: (*PluginServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetPluginInfo",
			Handler:    _PluginService_GetPluginInfo_Handler,
		},
		{
			MethodName: "ExecuteCommand",
			Handler:    _PluginService_ExecuteCommand_Handler,
		},
		{
			MethodName: "GetMenu",
			Handler:    _PluginService_GetMenu_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/plugin.proto",
}
