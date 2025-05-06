package main

import (
	"fmt"
	"sunny_5_skiers/sunny_5_skiers/core"
)

const (
	EventRegistration = iota + 1
	EventStartTimeSet
	EventOnStartLine
	EventStarted
	EventOnFiringRange
	EventShot
	EventLeftFiringRange
	EventEnteredPenalty
	EventLeftPenalty
	EventLapCompleted
	EventCannotContinue
	EventDisqualified = 32
	EventFinished     = 33
)

func main() {
	rs, err := core.NewRaceSystem("config.json")
	if err != nil {
		fmt.Println("Ошибка инициализации системы:", err)
		return
	}

	if err := rs.ProcessEventsFromFile("event"); err != nil {
		fmt.Println("Ошибка обработки событий:", err)
		return
	}

}
