package commands

import (
	"context"
	"fmt"
	"github.com/harryrose/godm/cli/queue"
	"github.com/urfave/cli/v3"
)

func ClearHistory() *cli.Command {
	return &cli.Command{
		Name:   "history",
		Usage:  "Clear a queue's finished items",
		Action: clearHistory,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  FlagQueue,
				Value: DefQueue,
			},
		},
	}
}

func clearHistory(ctx context.Context, cmd *cli.Command) error {
	client, err := getRPCClient(cmd)
	if err != nil {
		return err
	}
	_, err = client.ClearHistory(ctx, &queue.ClearHistoryInput{
		Queue: &queue.Identifier{
			Id: cmd.String(FlagQueue),
		},
	})
	if err != nil {
		return cli.Exit(fmt.Sprintf("error clearing history: %v", err), CodeInternalError)
	}
	return nil
}
