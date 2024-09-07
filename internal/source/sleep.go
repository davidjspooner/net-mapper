package source

import (
	"context"
	"time"
)

type Sleep struct {
	duration time.Duration
	timer    *time.Timer
}

func (s *Sleep) Wait(ctx context.Context) error {
	if s.duration == 0 {
		s.duration = 5 * time.Minute
	}

	if s.timer == nil {
		s.timer = time.NewTimer(s.duration)
	} else {
		s.timer.Reset(s.duration)
	}

	select {
	case <-ctx.Done():
		if !s.timer.Stop() {
			<-s.timer.C //drain
		}
		return ctx.Err()
	case <-s.timer.C:
		return nil
	}
}
