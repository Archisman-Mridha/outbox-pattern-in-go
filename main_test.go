package main

import (
	"context"
	"os/exec"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	sqlc_generated "github.com/Archisman-Mridha/message-dispatcher/adapters/dbs/sql/generated"
	protoc_generated "github.com/Archisman-Mridha/message-dispatcher/proto/generated"
	"github.com/Archisman-Mridha/message-dispatcher/utils"
)

func TestMessageDispatcher(t *testing.T) {

	// Execute 'docker-compose up -d --build' command to start Postgres, RabbitMQ and message-dispatcher
	// containers.
	command := exec.Command("docker-compose", "up", "-d", "--build")
	if err := command.Run( ); err != nil {
		t.Errorf("Error starting docker-compose project: %v", err)
	}
	// Give some time to the containers to be ready.
	time.Sleep(5 * time.Second)

	defer func( ) {
		command= exec.Command("docker-compose", "down")
		if err := command.Run( ); err != nil {
			t.Errorf("Error stopping docker-compose project: %v", err)
		}
	}( )

	const (
		BATCH_SIZE= 5
		DB_URI= "postgresql://default:password@localhost:5432/authentication?sslmode=disable"

		MQ_URI= "amqp://default:password@localhost:5672"
		MQ_QUEUE_NAME= "events-for-authentication-microservice"

		REGISTRATION_EMAIL= "archi.procoder@gmail.com"
	)

	t.Run("🧪 message-dispatcher should propagate the message from DB to mq successfully", func(t *testing.T) {

		// Establish connection with the outbox DB and MQ.
		var (
			dbConnection= utils.ConnectToPostgres(DB_URI)
			querier= sqlc_generated.New(dbConnection)

			mqConnection, mqChannel= utils.ConnectToRabbitMQ(MQ_URI, MQ_QUEUE_NAME)
		)
		defer func( ) {
			dbConnection.Close( )

			mqChannel.Close( )
			mqConnection.Close( )
		}( )

		// Insert a message into the outbox DB table.
		message, err := proto.Marshal(
			&protoc_generated.UserRegistrationStartedEvent{
				Email: REGISTRATION_EMAIL,
			},
		)
		if err != nil {
			t.Errorf("❌ Error proto marshalling: %v", err)
		}
		if err := querier.InsertMessage(context.Background( ), message); err != nil {
			t.Errorf("❌ Error inserting message into database: %v", err )
		}

		timeoutChan := make(chan bool)
		waitGroup := &sync.WaitGroup{ }
		waitGroup.Add(1)
		go func( ) {
			defer waitGroup.Done( )

			time.Sleep(30 * time.Second)
			timeoutChan <- true
		}( )

		// Wait for the message-dispatcher container to publish the message to the MQ. After the message
		// gets published, we will consume that message.
		messages, err := mqChannel.Consume(MQ_QUEUE_NAME, "", false, false, false, false, nil)
		if err != nil {
			t.Errorf("❌ Error trying to consume messages from RabbitMQ: %v", err)
		}
		select {
			case message := <- messages: {
				t.Log("✅ Message received in MQ. Test successfull!")
				unmarshalledMessage := &protoc_generated.UserRegistrationStartedEvent{ }
				if err := proto.Unmarshal(message.Body, unmarshalledMessage); err != nil {
					t.Errorf("❌ Error unmarshalling message from MQ: %v", err)
				}
				assert.Equal(t, unmarshalledMessage.Email, REGISTRATION_EMAIL)
			}

			// If the message doesn't reach the MQ within 30 seconds, then fail the test.
			case <- timeoutChan:
				t.Error("❌ Message didn't reach the MQ (timed out)")
		}

	})

}