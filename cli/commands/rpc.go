package commands

import (
	"context"
	"fmt"
	"github.com/harryrose/godm/cli/queue"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func getRPCClient(cliCtx *cli.Context) (queue.QueueServiceClient, error) {
	con, err := grpc.Dial(
		cliCtx.String(ArgQueueHost),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", cliCtx.String(ArgKey))
			return invoker(ctx, method, req, reply, cc, opts...)
		}),
	)
	if err != nil {
		return nil, cli.Exit(fmt.Sprintf("error connecting to queue host: %v", err), CodeNetworkError)
	}
	client := queue.NewQueueServiceClient(con)
	return client, nil
}
