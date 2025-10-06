package main

import (
	"context"
	"github.com/harryrose/godm/cli/commands"
	"github.com/urfave/cli/v3"
	"log"
	"os"
)

func main() {
	app := &cli.Command{
		Name:  "godm",
		Usage: "manage a GoDM queue",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     commands.ArgQueueHost,
				Aliases:  []string{"q"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     commands.ArgKey,
				Aliases:  []string{"k"},
				Required: true,
			},
		},
		Commands: []*cli.Command{
			commands.Add(),
			{
				Name:  "clear",
				Usage: "Remove all items from an object",
				Commands: []*cli.Command{
					commands.ClearHistory(),
				},
			},
			{
				Name: "show",
				Commands: []*cli.Command{
					commands.ShowQueue(),
					commands.ShowHistory(),
					commands.ShowQueues(),
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
