package main

import (
	"context"
	"fmt"
	"github.com/harryrose/godm/downloader"
	"github.com/harryrose/godm/downloader/queue"
	"github.com/harryrose/godm/downloader/size"
	"github.com/harryrose/godm/downloader/writer"
	"github.com/harryrose/godm/log"
	"github.com/harryrose/godm/log/keys"
	"github.com/harryrose/godm/log/levels"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"os"
	"strings"
	"time"
)

const (
	defaultQueue            = "default"
	defaultPollPeriod       = 10 * time.Second
	rateLimitBytesPerSecond = 1024 * 1024 * 10
)

const (
	EnvQueueAddress       = "GODM_D_QUEUE_ADDRESS"
	EnvConnectionTimeout  = "GODM_D_CONNECTION_TIMEOUT"
	EnvDownloadDirectory  = "GODM_D_DOWNLOAD_DIRECTORY"
	EnvKey                = "GODM_D_KEY"
	EnvUserAgent          = "GODM_D_USER_AGENT"
	EnvPollPeriod         = "GODM_D_POLL_PERIOD"
	EnvRateLimit          = "GODM_D_RATE_LIMIT"
	EnvQueue              = "GODM_D_QUEUE"
	FlagQueueAddress      = "queue-address"
	FlagConnectionTimeout = "connection-timeout"
	FlagDownloadDirectory = "download-directory"
	FlagKey               = "key"
	FlagUserAgent         = "user-agent"
	FlagPollPeriod        = "poll-period"
	FlagRateLimit         = "rate-limit"
	FlagQueue             = "queue"
)

func main() {
	if err := log.Init(levels.Info); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("unable to initialise logger: %w", err))
		os.Exit(1)
	}

	cmd := &cli.Command{
		Name:  "downloader",
		Usage: "A downloader for GoDM",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     FlagQueueAddress,
				Aliases:  []string{"a"},
				Usage:    "The address of the queue service",
				Required: true,
				Sources:  cli.NewValueSourceChain(cli.EnvVar(EnvQueueAddress)),
			},
			&cli.DurationFlag{
				Name:    FlagConnectionTimeout,
				Aliases: []string{"t"},
				Usage:   "The timeout for connecting to the queue service",
				Value:   10 * time.Second,
				Sources: cli.NewValueSourceChain(cli.EnvVar(EnvConnectionTimeout)),
			},
			&cli.StringFlag{
				Name:     FlagDownloadDirectory,
				Aliases:  []string{"d"},
				Usage:    "The directory to download files to",
				Required: true,
				Sources:  cli.NewValueSourceChain(cli.EnvVar(EnvDownloadDirectory)),
			},
			&cli.StringFlag{
				Name:     FlagKey,
				Aliases:  []string{"k"},
				Usage:    "The API key to use when connecting to the queue service",
				Required: true,
				Sources:  cli.NewValueSourceChain(cli.EnvVar(EnvKey)),
			},
			&cli.StringFlag{
				Name:    FlagUserAgent,
				Aliases: []string{"u"},
				Usage:   "The User-Agent string to use when downloading files",
				Value:   "GoDM/development",
				Sources: cli.NewValueSourceChain(cli.EnvVar(EnvUserAgent)),
			},
			&cli.DurationFlag{
				Name:    FlagPollPeriod,
				Aliases: []string{"p"},
				Usage:   "The period between polling the queue for new items to download",
				Value:   defaultPollPeriod,
				Sources: cli.NewValueSourceChain(cli.EnvVar(EnvPollPeriod)),
			},
			&size.Flag{
				Name:    FlagRateLimit,
				Aliases: []string{"r"},
				Usage:   "The maximum download rate in bytes per second (0 for unlimited). K, M, G suffixes are supported",
				Value:   size.Size(rateLimitBytesPerSecond),
				Sources: cli.NewValueSourceChain(cli.EnvVar(EnvRateLimit)),
			},
			&cli.StringFlag{
				Name:    FlagQueue,
				Aliases: []string{"q"},
				Usage:   "The queue to poll for items",
				Value:   defaultQueue,
				Sources: cli.NewValueSourceChain(cli.EnvVar(EnvQueue)),
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			downloadDir := command.String(FlagDownloadDirectory)
			if strings.TrimSpace(downloadDir) == "" {
				return fmt.Errorf("download directory cannot be empty")
			}
			ddstat, err := os.Stat(downloadDir)
			if os.IsNotExist(err) {
				return fmt.Errorf("download directory %v does not exist", downloadDir)
			} else if err != nil {
				return fmt.Errorf("error getting download directory stats %v: %v", downloadDir, err)
			}
			if !ddstat.IsDir() {
				return fmt.Errorf("download directory path, %v, is not a directory", downloadDir)
			}

			writer.ForceDownloadRoot(downloadDir)

			queueAddress := command.String(FlagQueueAddress)
			if queueAddress == "" {
				return fmt.Errorf("queue address cannot be empty")
			}
			key := command.String(FlagKey)
			if key == "" {
				return fmt.Errorf("key cannot be empty")
			}

			pollPeriod := command.Duration(FlagPollPeriod)
			if pollPeriod <= time.Second {
				return fmt.Errorf("poll period must be greater than 1 second")
			}

			queueName := command.String(FlagQueue)
			if strings.TrimSpace(queueName) == "" {
				return fmt.Errorf("queue cannot be empty")
			}

			rateLimit, ok := command.Value(FlagRateLimit).(size.Size)
			if !ok {
				return fmt.Errorf("rate limit was not a valid size")
			}
			if rateLimit < 0 {
				return fmt.Errorf("rate limit cannot be negative")
			}

			conn, err := grpc.Dial(
				queueAddress,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
					ctx = metadata.AppendToOutgoingContext(ctx, "authorization", key)
					return invoker(ctx, method, req, reply, cc, opts...)
				}),
			)

			if err != nil {
				log.Errorw("connection error", keys.Error, err, keys.Address, queueAddress)
				os.Exit(1)
			}

			client := queue.NewQueueServiceClient(conn)
			if err := testWriteAccessToDir(downloadDir); err != nil {
				log.Errorw("download dir error", keys.Path, downloadDir, keys.Error, err)
				os.Exit(1)
			}

			downloader.Run(context.Background(), client, pollPeriod, queueName, int(rateLimit.Bytes()))
			return nil
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Errorw("fatal error", keys.Error, err)
		os.Exit(1)
	}
}

func testWriteAccessToDir(dir string) error {
	statDir, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return fmt.Errorf("download directory %v does not exist", dir)
	}
	if err != nil {
		return fmt.Errorf("error getting download directory stats %v: %v", dir, err)
	}
	if !statDir.IsDir() {
		return fmt.Errorf("download directory path, %v, is not a directory", dir)
	}
	tmpFile, err := os.CreateTemp(dir, ".godm.*")
	if err != nil {
		return fmt.Errorf("unable to create temporary file")
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()
	_, err = fmt.Fprintln(tmpFile, "test of ability to write to this directory. this file can be safely deleted")
	if err != nil {
		return fmt.Errorf("unable to write to temporary file")
	}
	return nil
}
