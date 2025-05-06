package core

import (
	"time"
)

type Competitor struct {
	ID             int
	Registered     bool
	ScheduledStart time.Time
	ActualStart    time.Time
	Finished       bool
	Disqualified   bool
	LapsCompleted  int
	PenaltyLaps    int
	ShootingStats  [][][]bool
	TotalTime      time.Duration
	FirstEvent     time.Time
	LastEvent      time.Time
	EventsCount    int
}
