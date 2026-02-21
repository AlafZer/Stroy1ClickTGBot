package telegram

import (
	"Stroy1ClickBot/repository"
	"context"
	"log"
	"time"
)

type Cleaner struct {
	dur time.Duration
	db  *repository.Store
}

func NewCleaner(db *repository.Store, dur time.Duration) *Cleaner {
	return &Cleaner{
		dur: dur,
		db:  db,
	}
}

func (cln *Cleaner) Clean(ctx context.Context) error {
	timer := time.NewTicker(cln.dur)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			count, err := cln.db.CleanupExpiredTokens(ctx)
			if err != nil {
				return err
			}
			log.Printf("Cleaning up: %d tokens was destroyed", count)
		}
	}
}
