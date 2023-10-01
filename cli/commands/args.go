package commands

import (
	"github.com/urfave/cli/v2"
)

const (
	ArgCategory  = "category"
	ArgQueue     = "queue"
	ArgQueueHost = "queue-host"
	ArgKey       = "key"
)

const (
	DefCategory = "default"
	DefQueue    = "default"
)

func StringOrDefault(ctx *cli.Context, key string, defaultValue string) string {
	val := ctx.String(key)
	if val == "" {
		return defaultValue
	}
	return val
}
