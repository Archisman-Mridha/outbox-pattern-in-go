package ports

type (
	OutboxDB interface {
		// connect establishes a connection with the outbox DB.
		// connect(uri string)

		// Disconnect closes connection to the outbox DB.
		Disconnect( )

		// GetMessages queries the outbox database. This query searches for a batch of messages which are
		// unlocked and not yet published to the message queue. It then extracts the event payload from
		// each of those queried items. The event payloads are then sent inside the channel
		// args.toBePublishedItemsChan.
		GetMessages(args *GetMessagesArgs)

		// UnlockMessagesAndUpdatePublishStatus takes PublishResultsChan as an input. Transaction lock
		// is removed and the publish status is updated for each message.
		UnlockMessagesAndUpdatePublishStatus(args *UnlockMessagesAndUpdatePublishStatusArgs)
	
		// Clean cleans the database by deleting all the rows whichy correspond to those messages, which
		// have been published to the message queue.
		Clean( )
	}

	MQ interface {
		// Disconnect cleans up connection with the message queue.
		Disconnect( )

		// PublishMessages waits for messages in the toBePublishedItemsChan. It gets those messages and
		// publishes them to message queue.
		PublishMessages(args *PublishMessagesArgs)
	}
)

type (
	// ToBePublishedItem is a data structure which holds the message which needs to be published along
	// with the id of the corresponding DB row.
	ToBePublishedItem struct {
		RowId string
		Message []byte
	}
	// PublishResult is a data structure which represents whether the message with self.RowId was
	// successfully published or not.
	PublishResult struct  {
		RowId string
		IsPublished bool
	}

	GetMessagesArgs struct {
		BatchSize int
		ToBePublishedItemsChan chan *ToBePublishedItem
	}

	UnlockMessagesAndUpdatePublishStatusArgs struct {
		PublishResultsChan chan *PublishResult
	}

	PublishMessagesArgs struct {
		ToBePublishedItemsChan chan *ToBePublishedItem
		PublishResultsChan chan *PublishResult
	}
)