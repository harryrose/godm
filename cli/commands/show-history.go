package commands

import (
	"fmt"
	"github.com/harryrose/godm/cli/queue"
	"github.com/urfave/cli/v2"
	"os"
	"text/tabwriter"
)

func ShowHistory(ctx *cli.Context) error {
	client, err := getRPCClient(ctx)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 5, 2, 1, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "Source\tDestination\tState\tSize\tMessage")
	next := ""
	for {
		got, err := client.GetFinishedItems(ctx.Context, &queue.GetFinishedItemsInput{
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
			return cli.Exit(fmt.Sprintf("error fetching queue history items: %v", err), CodeInternalError)
		}
		next = got.Pagination.Next.Id
		for _, item := range got.Items {
			fmt.Fprintf(w, "\n%s\t%s\t%s\t%d\t%s", item.Item.Source.Url, item.Item.Destination.Url, stateToString(item.State.State), item.State.TotalSizeBytes, item.State.Message)
		}
		if next == "" {
			return nil
		}
	}
}

func stateToString(state queue.ItemState_State) string {
	switch state {
	case queue.ItemState_ITEM_STATE_DOWNLOADING:
		return "Downloading"
	case queue.ItemState_ITEM_STATE_COMPLETE:
		return "Complete"
	case queue.ItemState_ITEM_STATE_FAILED:
		return "Failed"
	case queue.ItemState_ITEM_STATE_UNSPECIFIED:
		return "Unspecified"
	default:
		return queue.ItemState_State_name[int32(state)]
	}
}
