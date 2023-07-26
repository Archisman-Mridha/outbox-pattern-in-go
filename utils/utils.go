package utils

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

// GetEnv tries to find the env with the given name in the underlying OS environment. If the env is
// not found, then it panics. If found, then the value of the env is returned.
func GetEnv(envName string) string {
	envValue, isEnvFound := os.LookupEnv(envName)
	if !isEnvFound {
		log.Panicf("❌ Env %s not found", envName)
	}

	return envValue
}

// RunFnPeriodically runs a given funcion (in a separate go-routine) periodically with the given
// time period. It also takes the 'done' channel as an input. Before exitting the program, close the
// done channel to cleanup resources.
func RunFnPeriodically[T interface{}](waitGroup *errgroup.Group, fn func(T), fnArgs T, period time.Duration) {
	waitGroup.Go(func( ) error {
		ticker := time.NewTicker(period)
		defer ticker.Stop( )

		for range ticker.C {
			fn(fnArgs)
		}

		return nil
	})
}

func ConnectPostgres(uri string) *sql.DB {
	connection, err := sql.Open("postgres", uri)
	if err != nil {
		log.Fatalf("❌ Error connecting to the database : %v", err)
	}
	if err := connection.Ping( ); err != nil {
		log.Fatalf("❌ Error pinging the database : %v", err)
	}

	log.Println("✅ Connected to Postgres")

	return connection
}

func ConnectRedis(options *redis.Options) (client *redis.Client) {
	client = redis.NewClient(options)

	if _, err := client.Ping( ).Result( ); err != nil {
		log.Fatalf("❌ Error connecting to Redis: %v", err)
	}
	if _, err := client.XGroupCreateMkStream("outbox", "outboxer", "0").Result( ); err != nil {
		log.Printf("❌ Error creating consumer group for the outbox Redis stream: %v", err)
	}

	log.Println("✅ Connected to Redis")

	return
}

func ConnectRabbitMQ(uri, queueName string) (*amqp.Connection, *amqp.Channel) {
	connection, err := amqp.Dial(uri)
	if err != nil {
		log.Panicf("❌ Error connecting to RabbitMQ: %v", err)
	}
	channel, err := connection.Channel( )
	if err != nil {
		log.Panicf("❌ Error creating channel in RabbitMQ")
	}

	_, err = channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Printf("❌ Error declaring queue %s in RabbitMQ: %v", queueName, err)
	}

	log.Println("✅ Connected to RabbitMQ")

	return connection, channel
}
