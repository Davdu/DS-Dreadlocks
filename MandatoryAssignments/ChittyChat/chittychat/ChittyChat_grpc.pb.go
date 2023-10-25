// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.24.4
// source: ChittyChat.proto

package chittychat

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	ChittyChat_Messages_FullMethodName = "/ChittyChat.ChittyChat/Messages"
)

// ChittyChatClient is the client API for ChittyChat service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ChittyChatClient interface {
	Messages(ctx context.Context, opts ...grpc.CallOption) (ChittyChat_MessagesClient, error)
}

type chittyChatClient struct {
	cc grpc.ClientConnInterface
}

func NewChittyChatClient(cc grpc.ClientConnInterface) ChittyChatClient {
	return &chittyChatClient{cc}
}

func (c *chittyChatClient) Messages(ctx context.Context, opts ...grpc.CallOption) (ChittyChat_MessagesClient, error) {
	stream, err := c.cc.NewStream(ctx, &ChittyChat_ServiceDesc.Streams[0], ChittyChat_Messages_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &chittyChatMessagesClient{stream}
	return x, nil
}

type ChittyChat_MessagesClient interface {
	Send(*Message) error
	Recv() (*Message, error)
	grpc.ClientStream
}

type chittyChatMessagesClient struct {
	grpc.ClientStream
}

func (x *chittyChatMessagesClient) Send(m *Message) error {
	return x.ClientStream.SendMsg(m)
}

func (x *chittyChatMessagesClient) Recv() (*Message, error) {
	m := new(Message)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ChittyChatServer is the server API for ChittyChat service.
// All implementations must embed UnimplementedChittyChatServer
// for forward compatibility
type ChittyChatServer interface {
	Messages(ChittyChat_MessagesServer) error
	mustEmbedUnimplementedChittyChatServer()
}

// UnimplementedChittyChatServer must be embedded to have forward compatible implementations.
type UnimplementedChittyChatServer struct {
}

func (UnimplementedChittyChatServer) Messages(ChittyChat_MessagesServer) error {
	return status.Errorf(codes.Unimplemented, "method Messages not implemented")
}
func (UnimplementedChittyChatServer) mustEmbedUnimplementedChittyChatServer() {}

// UnsafeChittyChatServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ChittyChatServer will
// result in compilation errors.
type UnsafeChittyChatServer interface {
	mustEmbedUnimplementedChittyChatServer()
}

func RegisterChittyChatServer(s grpc.ServiceRegistrar, srv ChittyChatServer) {
	s.RegisterService(&ChittyChat_ServiceDesc, srv)
}

func _ChittyChat_Messages_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ChittyChatServer).Messages(&chittyChatMessagesServer{stream})
}

type ChittyChat_MessagesServer interface {
	Send(*Message) error
	Recv() (*Message, error)
	grpc.ServerStream
}

type chittyChatMessagesServer struct {
	grpc.ServerStream
}

func (x *chittyChatMessagesServer) Send(m *Message) error {
	return x.ServerStream.SendMsg(m)
}

func (x *chittyChatMessagesServer) Recv() (*Message, error) {
	m := new(Message)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ChittyChat_ServiceDesc is the grpc.ServiceDesc for ChittyChat service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ChittyChat_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ChittyChat.ChittyChat",
	HandlerType: (*ChittyChatServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Messages",
			Handler:       _ChittyChat_Messages_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "ChittyChat.proto",
}
