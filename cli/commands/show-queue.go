package commands

import (
	"fmt"
	"github.com/harryrose/godm/cli/queue"
	"github.com/urfave/cli/v2"
	"os"
	"text/tabwriter"
)

func ShowQueue(ctx *cli.Context) error {
	client, err := getRPCClient(ctx)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 5, 2, 1, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "Source\tDestination\tDownloaded\tTotal\t%%")
	next := ""
	for {
		got, err := client.GetQueueItems(ctx.Context, &queue.GetQueueItemsInput{
			Queue: &queue.Identifier{
				Id: StringOrDefault(ctx, ArgQueue, DefQueue),
			},
			Pagination: &queue.PaginationParameters{
				Limit: 50,
				Next: &queue.Identifier{
					Id: next,
				},
			},
		})
		if err != nil {
			return cli.Exit(fmt.Sprintf("error fetching queue items: %v", err), CodeInternalError)
		}
		next = got.Pagination.Next.Id
		for _, item := range got.Items {
			perc := 0.0
			if item.State.TotalSizeBytes != 0 {
				perc = float64(item.State.DownloadedBytes) / float64(item.State.TotalSizeBytes)
			}
			fmt.Fprintf(w, "\n%s\t%s\t%d\t%d\t%4.1f", item.Item.Source.Url, item.Item.Destination.Url, item.State.DownloadedBytes, item.State.TotalSizeBytes, perc*100.0)
		}
		if next == "" {
			return nil
		}
	}
}
