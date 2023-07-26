package dbs

import (
	"log"

	"github.com/go-redis/redis"
	"github.com/google/uuid"

	"github.com/Archisman-Mridha/outboxer/domain/ports"
	"github.com/Archisman-Mridha/outboxer/utils"
)

type (
	RedisAdapter struct {
		client *redis.Client
		consumerName string
	}

	Record struct {
		ID int32
		Message []byte
		Locked bool
		Published bool
	}
)

func NewRedisAdapter(options *redis.Options) *RedisAdapter {
	r := &RedisAdapter{
		consumerName: uuid.NewString( ),
	}

	r.client= utils.ConnectRedis(options)

	return r
}

func (r *RedisAdapter) Disconnect( ) {
	if err := r.client.Close( ); err != nil {
		log.Fatalf("Error closing connection to Redis: %v", err)
	}
	log.Println("Closed connection to Redis")
}

func (r *RedisAdapter) GetMessages(args *ports.GetMessagesArgs) {
	// Fetch a batch of records from the Redis stream
	result, err := r.client.XReadGroup(&redis.XReadGroupArgs{
		Group: "outboxer",
		Consumer: r.consumerName,
		Streams: []string{ "outbox", ">" },
		Block: 1,
		Count: int64(args.BatchSize),
		NoAck: true,
	}).Result( )
	if err != nil {
		log.Printf("❌ Error retrieving messages from Redis stream: %v", err)
	}

	for _, item := range result {
		for _, item := range item.Messages {
			args.ToBePublishedItemsChan <- &ports.ToBePublishedItem{
				RowId: item.ID,
				Message: []byte(item.Values["message"].(string)),
			}
		}
	}
}

func (r *RedisAdapter) UnlockMessagesAndUpdatePublishStatus(args *ports.UnlockMessagesAndUpdatePublishStatusArgs) {
	for item := range args.PublishResultsChan {
		if item.IsPublished {
			if _, err := r.client.XAck("outbox", "default", item.RowId).Result( ); err != nil {
				log.Printf("❌ Error executing XAck Redis command: %v", err)
			}
		}
	}
}

func (r *RedisAdapter) Clean( ) { }