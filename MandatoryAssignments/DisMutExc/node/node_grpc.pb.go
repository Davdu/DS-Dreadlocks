// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.24.4
// source: node.Server

package __

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
	Node_Messages_FullMethodName = "/node.Node/Messages"
)

// NodeClient is the client API for Node service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type NodeClient interface {
	Messages(ctx context.Context, opts ...grpc.CallOption) (Node_MessagesClient, error)
}

type nodeClient struct {
	cc grpc.ClientConnInterface
}

func NewNodeClient(cc grpc.ClientConnInterface) NodeClient {
	return &nodeClient{cc}
}

func (c *nodeClient) Messages(ctx context.Context, opts ...grpc.CallOption) (Node_MessagesClient, error) {
	stream, err := c.cc.NewStream(ctx, &Node_ServiceDesc.Streams[0], Node_Messages_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &nodeMessagesClient{stream}
	return x, nil
}

type Node_MessagesClient interface {
	Send(*Message) error
	Recv() (*Message, error)
	grpc.ClientStream
}

type nodeMessagesClient struct {
	grpc.ClientStream
}

func (x *nodeMessagesClient) Send(m *Message) error {
	return x.ClientStream.SendMsg(m)
}

func (x *nodeMessagesClient) Recv() (*Message, error) {
	m := new(Message)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// NodeServer is the Server API for Node service.
// All implementations must embed UnimplementedNodeServer
// for forward compatibility
type NodeServer interface {
	Messages(Node_MessagesServer) error
	mustEmbedUnimplementedNodeServer()
}

// UnimplementedNodeServer must be embedded to have forward compatible implementations.
type UnimplementedNodeServer struct {
}

func (UnimplementedNodeServer) Messages(Node_MessagesServer) error {
	return status.Errorf(codes.Unimplemented, "method Messages not implemented")
}
func (UnimplementedNodeServer) mustEmbedUnimplementedNodeServer() {}

// UnsafeNodeServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to NodeServer will
// result in compilation errors.
type UnsafeNodeServer interface {
	mustEmbedUnimplementedNodeServer()
}

func RegisterNodeServer(s grpc.ServiceRegistrar, srv NodeServer) {
	s.RegisterService(&Node_ServiceDesc, srv)
}

func _Node_Messages_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(NodeServer).Messages(&nodeMessagesServer{stream})
}

type Node_MessagesServer interface {
	Send(*Message) error
	Recv() (*Message, error)
	grpc.ServerStream
}

type nodeMessagesServer struct {
	grpc.ServerStream
}

func (x *nodeMessagesServer) Send(m *Message) error {
	return x.ServerStream.SendMsg(m)
}

func (x *nodeMessagesServer) Recv() (*Message, error) {
	m := new(Message)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Node_ServiceDesc is the grpc.ServiceDesc for Node service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Node_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "node.Node",
	HandlerType: (*NodeServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Messages",
			Handler:       _Node_Messages_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "node.Server",
}
