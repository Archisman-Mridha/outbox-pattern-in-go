package mqs

import (
	"log"

	"github.com/streadway/amqp"

	"github.com/Archisman-Mridha/outboxer/domain/ports"
	"github.com/Archisman-Mridha/outboxer/utils"
)

type RabbitMQAdapter struct {
	connection *amqp.Connection
	channel *amqp.Channel
	queueName string
}

func NewRabbitMQAdapter(uri, queueName string) *RabbitMQAdapter {
	r := &RabbitMQAdapter{ queueName: queueName }
	r.connection, r.channel= utils.ConnectRabbitMQ(uri, queueName)

	return r
}

func(r *RabbitMQAdapter) Disconnect( ) {
	if err := r.connection.Close( ); err != nil {
		log.Printf("❌ Error closing connection to the database: %v", err)
	}
}

func(r *RabbitMQAdapter) PublishMessages(args *ports.PublishMessagesArgs) {
	for item := range args.ToBePublishedItemsChan {
		err := r.channel.Publish("", r.queueName, true, false, amqp.Publishing{ Body: item.Message })
		if err != nil {
			log.Printf("❌ Error trying to publish message to rabbitMQ: %v", err)
		}
		args.PublishResultsChan <- &ports.PublishResult{
			RowId: item.RowId,
			IsPublished: err == nil,
		}
	}
}