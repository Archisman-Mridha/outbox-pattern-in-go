package usecases

import (
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/Archisman-Mridha/outboxer/domain/ports"
	"github.com/Archisman-Mridha/outboxer/utils"
)

type RunArgs struct {
	WaitGroup *errgroup.Group

	OutboxDB ports.OutboxDB
	BatchSize int

	MQ ports.MQ
}

func(u *Usecases) Run(args RunArgs) {
	var (
		tobePublishedItemsChan= make(chan *ports.ToBePublishedItem)
		publishResultsChan= make(chan *ports.PublishResult)
	)

	utils.RunFnPeriodically[*ports.GetMessagesArgs](
		args.WaitGroup,
		args.OutboxDB.GetMessages,
		&ports.GetMessagesArgs{
			BatchSize: args.BatchSize,
			ToBePublishedItemsChan: tobePublishedItemsChan,
		},
		3 * time.Second,
	)

	args.WaitGroup.Go(func( ) error {
		args.MQ.PublishMessages(&ports.PublishMessagesArgs{
			ToBePublishedItemsChan: tobePublishedItemsChan,
			PublishResultsChan: publishResultsChan,
		})

		return nil
	})

	args.WaitGroup.Go(func( ) error {
		args.OutboxDB.Clean( )

		return nil
	})
}