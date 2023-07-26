package main

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-redis/redis"
	_ "github.com/lib/pq"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	"github.com/Archisman-Mridha/outboxer/adapters/dbs"
	"github.com/Archisman-Mridha/outboxer/adapters/mqs"
	"github.com/Archisman-Mridha/outboxer/domain/ports"
	"github.com/Archisman-Mridha/outboxer/domain/usecases"
)

func main( ) {
	configFileData, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Panicf("Error reading file config.yaml: %v", err)
	}
	var config Config
	if err= yaml.Unmarshal(configFileData, &config); err != nil {
		log.Panicf("Error unmarshalling config: %v", err)
	}

	waitGroup, waitGroupContext := errgroup.WithContext(context.Background( ))
	// Listen for system interruption signals to gracefully shut down
	waitGroup.Go(func( ) error {
		shutdownSignalChan := make(chan os.Signal, 1)
		signal.Notify(shutdownSignalChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(shutdownSignalChan)

		var err error

		select {
			case <- waitGroupContext.Done( ):
				err= waitGroupContext.Err( )

			case shutdownSignal := <- shutdownSignalChan:
				log.Printf("Received program shutdown signal %v", shutdownSignal)
				err= errors.New("❌ Received program interruption signal")
		}

		return err
	})

	var mq ports.MQ= mqs.NewRabbitMQAdapter(config.Sink.Uri, config.Sink.Queue)
	defer mq.Disconnect( )

	usecasesLayer := &usecases.Usecases{ }

	if config.Sources.Postgres != nil {
		var outboxDB ports.OutboxDB= dbs.NewPostgresAdapter(config.Sources.Postgres.Uri)
		defer outboxDB.Disconnect( )

		usecasesLayer.Run(usecases.RunArgs{
			WaitGroup: waitGroup,

			OutboxDB: outboxDB,
			BatchSize: config.Sources.Postgres.BatchSize,

			MQ: mq,
		})
	}

	if config.Sources.Redis != nil {
		var outboxDB ports.OutboxDB= dbs.NewRedisAdapter(&redis.Options{
			Addr: config.Sources.Redis.Uri,
			Password: config.Sources.Redis.Password,
		})
		defer outboxDB.Disconnect( )

		usecasesLayer.Run(usecases.RunArgs{
			WaitGroup: waitGroup,

			OutboxDB: outboxDB,
			BatchSize: config.Sources.Redis.BatchSize,

			MQ: mq,
		})
	}

	waitGroup.Wait( )
}