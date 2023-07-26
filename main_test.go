package main

import (
	"context"
	"log"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	sqlc_generated "github.com/Archisman-Mridha/outboxer/adapters/dbs/sql/generated"
	protoc_generated "github.com/Archisman-Mridha/outboxer/proto/generated"
	"github.com/Archisman-Mridha/outboxer/utils"
)

const (
	BATCH_SIZE= 5
	POSTGRES_URI= "postgresql://default:password@localhost:5432/authentication?sslmode=disable"
	REDIS_URI= "localhost:6379"
	REDIS_PASSWORD= "password"

	MQ_URI= "amqp://default:password@localhost:5672"
	MQ_QUEUE_NAME= "for-authentication-microservice"

	REGISTRATION_EMAIL= "archi.procoder@gmail.com"
	REGISTRATION_USERNAME= "archi"
)

func TestMessageDispatcher(t *testing.T) {

	// Execute 'docker-compose up -d --build' command to start Postgres, RabbitMQ and outboxer
	// containers.
	command := exec.Command("docker-compose", "up", "--build", "-d")
	if err := command.Run( ); err != nil {
		t.Errorf("Error starting docker-compose project: %v", err)
	}
	// Give some time to the containers to be ready.
	time.Sleep(10 * time.Second)

	defer func( ) {
		command= exec.Command("docker-compose", "down")
		if err := command.Run( ); err != nil {
			t.Errorf("Error stopping docker-compose project: %v", err)
		}
	}( )

	// Establish connection with RabbitMQ.
	var mqConnection, mqChannel = utils.ConnectRabbitMQ(MQ_URI, MQ_QUEUE_NAME)
	defer func( ) {
		mqChannel.Close( )
		mqConnection.Close( )
	}( )

	// This message will be inserted in the database and then the outboxer component will propagate it
	// to the message queue.
	message, err := proto.Marshal(
		&protoc_generated.RegistrationStartedEvent{
			Email: REGISTRATION_EMAIL,
			Username: REGISTRATION_USERNAME,
		},
	)
	if err != nil {
		t.Errorf("❌ Error proto marshalling: %v", err)
	}

	t.Run("🧪 outboxer should propagate the message from Redis to RabbitMQ successfully", func(t *testing.T) {
		testFnTemplate(t, "redis", mqChannel,
			func( ) {
				client := utils.ConnectRedis(&redis.Options{
					Addr: REDIS_URI,
					Password: REDIS_PASSWORD,
				})

				_, err := client.XAdd(&redis.XAddArgs{
					Stream: "outbox",
					Values: map[string]interface{}{ "message": message },
				}).Result( )
				if err != nil {
					log.Fatalf("❌ Error inserting data into redis stream: %v", err)
				}
			},
		)
	})

	t.Run("🧪 outboxer should propagate the message from Postgres to RabbitMQ successfully", func(t *testing.T) {
		postgresConnection := utils.ConnectPostgres(POSTGRES_URI)
		postgresQuerier := sqlc_generated.New(postgresConnection)

		if err := postgresQuerier.InsertMessage(context.Background( ), message); err != nil {
			t.Errorf("❌ Error inserting message into database: %v", err )
		}

		postgresConnection.Close( )
	})

}

func testFnTemplate(t *testing.T, dbType string, mqChannel *amqp.Channel, insertMessageIntoDB func( )) {
	insertMessageIntoDB( )

	timeoutChan := make(chan bool)
	waitGroup := &sync.WaitGroup{ }
	waitGroup.Add(1)
	go func( ) {
		defer waitGroup.Done( )

		time.Sleep(30 * time.Second)
		timeoutChan <- true
	}( )

	// Wait for the outboxer container to publish the message to the MQ. After the message
	// gets published, we will consume that message.
	messages, err := mqChannel.Consume(MQ_QUEUE_NAME, "", true, false, false, false, nil)
	if err != nil {
		t.Errorf("❌ Error trying to consume messages from RabbitMQ: %v", err)
	}
	select {
		case message := <- messages: {
			t.Logf("✅ Message received in MQ. Test successfull for %s!", dbType)
			unmarshalledMessage := &protoc_generated.RegistrationStartedEvent{ }
			if err := proto.Unmarshal(message.Body, unmarshalledMessage); err != nil {
				t.Errorf("❌ Error unmarshalling message from MQ: %v", err)
			}
			assert.Equal(t, unmarshalledMessage.Email, REGISTRATION_EMAIL)
			assert.Equal(t, unmarshalledMessage.Username, REGISTRATION_USERNAME)

			waitGroup.Done( )
			return
		}

		// If the message doesn't reach the MQ within 10 seconds, then fail the test.
		case <- timeoutChan:
			t.Error("❌ Message didn't reach the MQ (timed out)")
	}
}