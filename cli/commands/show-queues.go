package commands

import (
	"context"
	"fmt"
	"github.com/harryrose/godm/cli/queue"
	"github.com/urfave/cli/v3"
	"os"
)

func ShowQueues() *cli.Command {
	return &cli.Command{
		Name:   "queues",
		Usage:  "Show all queues",
		Action: showQueues,
	}
}

func showQueues(ctx context.Context, cmd *cli.Command) error {
	client, err := getRPCClient(cmd)
	if err != nil {
		return err
	}
	qs, err := client.ListQueues(ctx, &queue.ListQueuesInput{})
	if err != nil {
		return cli.Exit("error fetching queues", CodeInternalError)
	}
	for _, q := range qs.Queues {
		fmt.Fprintf(os.Stdout, "%s\n", q.Name)
	}
	return nil
}
