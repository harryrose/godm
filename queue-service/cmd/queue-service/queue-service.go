package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/harryrose/godm/log"
	"github.com/harryrose/godm/log/levels"
	"github.com/harryrose/godm/queue-service"
	"github.com/harryrose/godm/queue-service/auth"
	"github.com/harryrose/godm/queue-service/db"
	"github.com/harryrose/godm/queue-service/rpc"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc"
	golog "log"
	"net"
	"os"
)

const (
	keyLengthBytes = 16

	FlagPort  = "port"
	FlagDB    = "database"
	FlagKey   = "key"
	FlagQueue = "queue"

	EnvPort  = "GODM_Q_PORT"
	EnvDB    = "GODM_Q_DATABASE"
	EnvKey   = "GODM_Q_KEY"
	EnvQueue = "GODM_Q_QUEUE"
)

func main() {
	if err := log.Init(levels.Info); err != nil {
		golog.Fatalf("unable to initialise logger: %w", err)
	}

	cmd := &cli.Command{
		Name:  "queue-service",
		Usage: "A queue service for GoDM",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    FlagPort,
				Aliases: []string{"p"},
				Usage:   "The port to listen on",
				Value:   9010,
				Sources: cli.NewValueSourceChain(cli.EnvVar(EnvPort)),
			},
			&cli.StringFlag{
				Name:    FlagDB,
				Aliases: []string{"d"},
				Usage:   "The path to the database file",
				Value:   "queue.db",
				Sources: cli.NewValueSourceChain(cli.EnvVar(EnvDB)),
			},
			&cli.StringFlag{
				Name:    FlagKey,
				Aliases: []string{"k"},
				Usage:   "The base64-encoded key to use for authentication. If not provided, a random key will be generated.",
				Value:   "",
				Sources: cli.NewValueSourceChain(cli.EnvVar(EnvKey)),
			},
			&cli.StringFlag{
				Name:    FlagQueue,
				Aliases: []string{"q"},
				Usage:   "The name of the default queue to create",
				Value:   "default",
				Sources: cli.NewValueSourceChain(cli.EnvVar(EnvQueue)),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			key := cmd.String(FlagKey)
			if len(key) == 0 {
				// generate a random key
				buf := make([]byte, keyLengthBytes)
				if _, err := rand.Read(buf); err != nil {
					return fmt.Errorf("unable to read random data: %w", err)
				}
				key = base64.RawURLEncoding.EncodeToString(buf)
				log.Infow("generated key", "key", key)
			}

			port := cmd.Int(FlagPort)
			listenString := fmt.Sprintf(":%d", port)
			listener, err := net.Listen("tcp", listenString)
			if err != nil {
				return err
			}
			dbPath := cmd.String(FlagDB)
			database, err := db.NewBolt(dbPath)
			if err != nil {
				return err
			}
			svc := queue_service.Service{DB: database}

			queue := cmd.String(FlagQueue)
			_, err = database.CreateQueue(queue)
			if err != nil && !errors.As(err, &db.ErrConflict{}) {
				return err
			}

			grpcServer := grpc.NewServer(
				grpc.UnaryInterceptor(auth.AuthorizationInterceptor(key)),
			)

			log.Infow("listening", "address", listenString)
			rpc.RegisterQueueServiceServer(grpcServer, &svc)
			if err := grpcServer.Serve(listener); err != nil {
				return fmt.Errorf("error serving grpc: %w", err)
			}
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Errorw("error running command", "error", err)
		os.Exit(1)
	}
	log.Infow("exited")
}
