package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/harryrose/godm/log"
	"github.com/harryrose/godm/log/keys"
	"github.com/harryrose/godm/log/levels"
	"github.com/harryrose/godm/queue-service"
	"github.com/harryrose/godm/queue-service/auth"
	"github.com/harryrose/godm/queue-service/db"
	"github.com/harryrose/godm/queue-service/rpc"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	golog "log"
	"net"
	"os"
)

const (
	keyLengthBytes = 16
)

var Cfg struct {
	Port int    `required:"true" default:"9010"`
	DB   string `required:"true" default:"queue.db"`
	Key  string
}

func main() {
	envconfig.MustProcess("GODM_Q", &Cfg)
	if err := log.Init(levels.Info); err != nil {
		golog.Fatalf("unable to initialise logger: %w", err)
	}

	if len(Cfg.Key) == 0 {
		// generate a random key
		buf := make([]byte, keyLengthBytes)
		if _, err := rand.Read(buf); err != nil {
			log.Errorw("unable to read random data", keys.Error, err)
			os.Exit(1)
		}
		Cfg.Key = base64.RawURLEncoding.EncodeToString(buf)
		log.Infow("generated key", "key", Cfg.Key)
	}

	listenString := fmt.Sprintf(":%d", Cfg.Port)
	listener, err := net.Listen("tcp", listenString)
	if err != nil {
		panic(err)
	}
	database, err := db.NewBolt(Cfg.DB)
	if err != nil {
		panic(err)
	}
	svc := queue_service.Service{DB: database}

	_, err = database.CreateQueue("default")
	if err != nil && !errors.As(err, &db.ErrConflict{}) {
		panic(err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.AuthorizationInterceptor(Cfg.Key)),
	)

	log.Infow("listening", "address", listenString)
	rpc.RegisterQueueServiceServer(grpcServer, &svc)
	if err := grpcServer.Serve(listener); err != nil {
		log.Errorw("error serving grpc", keys.Error, err)
		os.Exit(1)
	}
	log.Infow("exited")
}
