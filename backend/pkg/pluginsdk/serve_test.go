package pluginsdk

import (
	"context"
	"net"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pluginv1 "xiaoheiplay/plugin/v1"
)

// echoService is a minimal in-process unary service used to exercise the
// gRPC server created by grpcServerFactory. GetMonitorResponse is reused as
// both request and response only because it carries a large string field.
type echoService struct {
	respSize int
}

func (e *echoService) handle(_ context.Context, _ *pluginv1.GetMonitorResponse) (*pluginv1.GetMonitorResponse, error) {
	if e.respSize > 0 {
		return &pluginv1.GetMonitorResponse{RawJson: strings.Repeat("x", e.respSize)}, nil
	}
	return &pluginv1.GetMonitorResponse{}, nil
}

var echoServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.EchoService",
	HandlerType: (*interface{})(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Echo",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(pluginv1.GetMonitorResponse)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(*echoService).handle(ctx, in)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/test.EchoService/Echo"}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(*echoService).handle(ctx, req.(*pluginv1.GetMonitorResponse))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
	Streams: []grpc.StreamDesc{},
}

func startEchoServer(t *testing.T, svc *echoService) (grpc.ClientConnInterface, func()) {
	t.Helper()
	s := grpcServerFactory(nil)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s.RegisterService(&echoServiceDesc, svc)
	go func() { _ = s.Serve(lis) }()
	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// raise the client recv cap so a failure in the send-limit test is
		// attributable to the server's MaxSendMsgSize, not the client default.
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(256<<20)),
	)
	if err != nil {
		t.Fatal(err)
	}
	return conn, func() {
		_ = conn.Close()
		s.Stop()
		_ = lis.Close()
	}
}

// TestServeGRPCServerEnforcesRecvLimit guards the plugin-side receive bound
// (audit M-1): requests larger than maxGRPCMessageSize must be rejected with
// ResourceExhausted instead of being accepted into the plugin process.
func TestServeGRPCServerEnforcesRecvLimit(t *testing.T) {
	conn, cleanup := startEchoServer(t, &echoService{})
	defer cleanup()

	big := &pluginv1.GetMonitorResponse{RawJson: strings.Repeat("x", maxGRPCMessageSize+1)}
	err := conn.Invoke(context.Background(), "/test.EchoService/Echo", big, new(pluginv1.GetMonitorResponse))
	if err == nil {
		t.Fatal("expected error for request exceeding MaxRecvMsgSize")
	}
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted, got %v (%v)", status.Code(err), err)
	}
}

// TestServeGRPCServerAcceptsRequestUpToLimit guards the raised receive bound:
// the gRPC server default is 4 MiB, which rejected legitimate host->plugin
// requests (e.g. large config_json); requests within the 64 MiB limit must
// succeed.
func TestServeGRPCServerAcceptsRequestUpToLimit(t *testing.T) {
	conn, cleanup := startEchoServer(t, &echoService{})
	defer cleanup()

	ok := &pluginv1.GetMonitorResponse{RawJson: strings.Repeat("x", 10<<20)} // 10 MiB, within limit
	var out pluginv1.GetMonitorResponse
	if err := conn.Invoke(context.Background(), "/test.EchoService/Echo", ok, &out); err != nil {
		t.Fatalf("request within MaxRecvMsgSize should succeed, got %v", err)
	}
}

// TestServeGRPCServerEnforcesSendLimit guards the plugin-side send bound
// (audit M-1): responses larger than maxGRPCMessageSize must be rejected at the
// plugin side so a defective plugin cannot push a multi-GiB payload into the
// host process.
func TestServeGRPCServerEnforcesSendLimit(t *testing.T) {
	conn, cleanup := startEchoServer(t, &echoService{respSize: maxGRPCMessageSize + 1})
	defer cleanup()

	err := conn.Invoke(context.Background(), "/test.EchoService/Echo", new(pluginv1.GetMonitorResponse), new(pluginv1.GetMonitorResponse))
	if err == nil {
		t.Fatal("expected error for response exceeding MaxSendMsgSize")
	}
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted, got %v (%v)", status.Code(err), err)
	}
}
