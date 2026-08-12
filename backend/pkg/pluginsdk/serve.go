package pluginsdk

import (
	"google.golang.org/grpc"

	"github.com/hashicorp/go-plugin"
)

type Plugin = plugin.Plugin

// maxGRPCMessageSize bounds plugin-side gRPC messages in both directions.
// go-plugin's client dials with MaxCallRecvMsgSize/MaxCallSendMsgSize set to
// math.MaxInt32 (~2 GiB), so without an explicit bound a defective or
// compromised plugin could push a multi-GiB response into the host process
// (OOM). The gRPC server default receive limit is 4 MiB, which also rejects
// legitimate larger host->plugin requests. 64 MiB is a generous cap for
// control-plane plugin RPCs.
const maxGRPCMessageSize = 64 << 20

func grpcServerFactory(opts []grpc.ServerOption) *grpc.Server {
	opts = append(opts,
		grpc.MaxRecvMsgSize(maxGRPCMessageSize),
		grpc.MaxSendMsgSize(maxGRPCMessageSize),
	)
	return grpc.NewServer(opts...)
}

func Serve(plugins map[string]Plugin) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         plugins,
		GRPCServer:      grpcServerFactory,
	})
}
