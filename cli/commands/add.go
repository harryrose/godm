package commands

import (
	"context"
	"fmt"
	"github.com/harryrose/godm/cli/queue"
	"github.com/urfave/cli/v3"
	"net/url"
	"os"
)

const (
	ArgSourceURL       = "source_url"
	ArgDestinationPath = "destination_path"
)

func Add() *cli.Command {
	return &cli.Command{
		Name:   "add",
		Usage:  "Queue an item for download",
		Action: add,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        FlagQueue,
				DefaultText: DefQueue,
			},
			&cli.StringFlag{
				Name:        FlagCategory,
				Aliases:     []string{"cat"},
				DefaultText: DefCategory,
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      ArgSourceURL,
				Value:     "",
				UsageText: "The URL to download from",
			},
			&cli.StringArg{
				Name:      ArgDestinationPath,
				Value:     "",
				UsageText: "The path to download the file to. Note that this is relative to the downloader's path.",
			},
		},
		ArgsUsage: "<source_url> <destination_path>",
	}
}

func add(ctx context.Context, cmd *cli.Command) error {

	srcStr := cmd.StringArg(ArgSourceURL)
	dstStr := cmd.StringArg(ArgDestinationPath)

	if srcStr == "" || dstStr == "" {
		return cli.Exit("source_url and destination_path are required", CodeInvalidArgument)
	}

	srcUrl, err := url.Parse(srcStr)
	if err != nil {
		return cli.Exit(fmt.Sprintf("source url was invalid: %v", err), CodeInvalidArgument)
	}
	if len(srcUrl.Scheme) == 0 {
		return cli.Exit(fmt.Sprintf("source url is missing a scheme: %v", err), CodeInvalidArgument)
	}

	dstUrl, err := url.Parse(dstStr)
	if err != nil {
		return cli.Exit(fmt.Sprintf("destination url is invalid: %v", err), CodeInvalidArgument)
	}
	if len(dstUrl.Scheme) == 0 {
		dstUrl.Scheme = "file"
	} else if dstUrl.Scheme != "file" {
		return cli.Exit("only file:// destination urls are supported", CodeInvalidArgument)
	}

	client, err := getRPCClient(cmd)
	if err != nil {
		return err
	}
	_, err = client.EnqueueItem(ctx, &queue.EnqueueItemInput{
		Queue: &queue.Identifier{Id: cmd.String(FlagQueue)},
		Item: &queue.Item{
			Source:      &queue.Target{Url: srcUrl.String()},
			Destination: &queue.Target{Url: dstUrl.String()},
			Category:    &queue.Category{Id: &queue.Identifier{Id: cmd.String(FlagCategory)}},
		},
	})
	if err != nil {
		return cli.Exit(fmt.Sprintf("error adding the item to the queue: %v", err), CodeInternalError)
	}
	fmt.Fprintln(os.Stderr, "item added")
	return nil
}
