package commands

import (
	"fmt"
	"github.com/harryrose/godm/cli/queue"
	"github.com/urfave/cli/v2"
)

func ClearHistory(ctx *cli.Context) error {
	client, err := getRPCClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.ClearHistory(ctx.Context, &queue.ClearHistoryInput{
		Queue: &queue.Identifier{
			Id: StringOrDefault(ctx, ArgQueue, DefQueue),
		},
	})
	if err != nil {
		return cli.Exit(fmt.Sprintf("error clearing history: %v", err), CodeInternalError)
	}
	return nil
}
