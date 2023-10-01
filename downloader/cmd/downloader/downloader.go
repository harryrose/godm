package main

import (
	"context"
	"fmt"
	"github.com/harryrose/godm/downloader"
	"github.com/harryrose/godm/downloader/queue"
	"github.com/harryrose/godm/downloader/writer"
	"github.com/harryrose/godm/log"
	"github.com/harryrose/godm/log/keys"
	"github.com/harryrose/godm/log/levels"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	golog "log"
	"os"
	"time"
)

const (
	defaultQueue            = "default"
	pollPeriod              = 10 * time.Second
	rateLimitBytesPerSecond = 1024 * 1024 * 10
)

type Config struct {
	QueueAddress      string `split_words:"true" required:"true"`
	ConnectionTimeout string `split_words:"true" default:"10s" required:"true"`
	DownloadDirectory string `split_words:"true" required:"true"`
	Key               string `required:"true"`
	UserAgent         string `split_words:"true" default:"GoDM/development"`
}

func main() {
	var cfg Config
	envconfig.MustProcess("GODM_D", &cfg)

	if err := log.Init(levels.Info); err != nil {
		golog.Fatalf("unable to initialise logger: %v", err)
	}

	writer.ForceDownloadRoot(cfg.DownloadDirectory)

	conn, err := grpc.Dial(
		cfg.QueueAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", cfg.Key)
			return invoker(ctx, method, req, reply, cc, opts...)
		}),
	)

	if err != nil {
		log.Errorw("connection error", keys.Error, err, keys.Address, cfg.QueueAddress)
		os.Exit(1)
	}

	client := queue.NewQueueServiceClient(conn)
	if err := testWriteAccessToDir(cfg.DownloadDirectory); err != nil {
		log.Errorw("download dir error", keys.Path, cfg.DownloadDirectory, keys.Error, err)
		os.Exit(1)
	}

	downloader.Run(context.Background(), client, pollPeriod, defaultQueue, rateLimitBytesPerSecond)
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
