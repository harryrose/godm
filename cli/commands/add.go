package commands

import (
	"fmt"
	"github.com/harryrose/godm/cli/queue"
	"github.com/urfave/cli/v2"
	"net/url"
	"os"
)

func Add(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return cli.Exit("expected two arguments -- the url to fetch from and a path to store to", CodeInvalidArgument)
	}
	srcStr := ctx.Args().Get(0)
	dstStr := ctx.Args().Get(1)

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

	client, err := getRPCClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.EnqueueItem(ctx.Context, &queue.EnqueueItemInput{
		Queue: &queue.Identifier{Id: StringOrDefault(ctx, ArgQueue, DefQueue)},
		Item: &queue.Item{
			Source:      &queue.Target{Url: srcUrl.String()},
			Destination: &queue.Target{Url: dstUrl.String()},
			Category:    &queue.Category{Id: &queue.Identifier{Id: StringOrDefault(ctx, ArgCategory, DefCategory)}},
		},
	})
	if err != nil {
		return cli.Exit(fmt.Sprintf("error adding the item to the queue: %v", err), CodeInternalError)
	}
	fmt.Fprintln(os.Stderr, "item added")
	return nil
}
