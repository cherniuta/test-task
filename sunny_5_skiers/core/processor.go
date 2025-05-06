package core

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"sunny_5_skiers/sunny_5_skiers/config"
	"time"
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

type RaceSystem struct {
	Config      *config.Config
	Competitors map[int]*Competitor
	CurrentLap  int
	logger      Logger
}

type Logger interface {
	LogEvent(timeStr, event string)
}

type ConsoleLogger struct{}

func (l ConsoleLogger) LogEvent(timeStr string, event string) {
	fmt.Println("["+timeStr+"] ", event)
}

func NewRaceSystem(configPath string) (*RaceSystem, error) {
	config, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	return &RaceSystem{
		Config:      config,
		Competitors: make(map[int]*Competitor),
		CurrentLap:  0,
		logger:      ConsoleLogger{},
	}, nil
}

func (rs *RaceSystem) ProcessEvent(timeStr string, eventID, competitorID int, params string) error {
	t, err := time.Parse("15:04:05.000", timeStr)
	if err != nil {
		return fmt.Errorf("неверный формат времени: %w", err)
	}

	comp := rs.getCompetitor(competitorID)

	switch eventID {
	case EventRegistration:
		return rs.handleRegistration(timeStr, comp)
	case EventStartTimeSet:
		return rs.handleStartTimeSet(comp, params, timeStr)
	case EventOnStartLine:
		return nil
	case EventStarted:
		return rs.handleStart(comp, t, timeStr)
	case EventOnFiringRange:
		rangeNum, err := parseInt(params)
		if err != nil {
			return err
		}
		return rs.handleFiringRangeEntry(comp, rangeNum, timeStr)
	case EventShot:
		target, err := parseInt(params)
		if err != nil {
			return err
		}
		return rs.processShot(comp, target, timeStr)
	case EventLeftFiringRange:
		return rs.processRangeExit(comp, timeStr)
	case EventEnteredPenalty:
		return rs.processPenaltyEntry(comp, timeStr)
	case EventLeftPenalty:
		return rs.processPenaltyExit(comp, timeStr)
	case EventLapCompleted:
		return rs.processLapCompletion(comp, t, timeStr)
	case EventCannotContinue:
		return rs.disqualify(comp, "NotFinished: "+params, timeStr)
	default:
		return fmt.Errorf("неизвестный идентификатор события: %d", eventID)
	}
}
func (rs *RaceSystem) handleRegistration(timeStr string, comp *Competitor) error {
	if comp.Registered {
		return fmt.Errorf("The competitor(%d) registered", comp.ID)
	}
	comp.Registered = true
	rs.logger.LogEvent(timeStr, fmt.Sprintf("The competitor(%d) registered", comp.ID))
	return nil
}

// handleStartTimeSet устанавливает время старта
func (rs *RaceSystem) handleStartTimeSet(comp *Competitor, params string, timeStr string) error {
	t, err := time.Parse("15:04:05.000", params)
	if err != nil {
		return fmt.Errorf("неверный формат времени старта: %w", err)
	}
	comp.ScheduledStart = t
	rs.logger.LogEvent(timeStr, fmt.Sprintf("The start time for the competitor(%d) was set by a draw to %s", comp.ID, t.Format("15:04:05.000")))
	return nil
}

func (rs *RaceSystem) handleStart(comp *Competitor, startTime time.Time, timeStr string) error {
	comp.ActualStart = startTime
	rs.logger.LogEvent(timeStr, fmt.Sprintf("The competitor(%d) has started", comp.ID))

	timeInterval, _ := time.Parse("15:04:05", rs.Config.StartDelta)

	startInterval := time.Duration(
		timeInterval.Hour()*int(time.Hour) +
			timeInterval.Minute()*int(time.Minute) +
			timeInterval.Second()*int(time.Second),
	)

	maxAllowedStart := comp.ScheduledStart.Add(startInterval)

	if startTime.After(maxAllowedStart) {
		return rs.disqualify(comp, fmt.Sprintf("опоздание на старт (допустимо до %s)", maxAllowedStart.Format("15:04:05.000")), timeStr)
	}
	return nil
}

func (rs *RaceSystem) handleFiringRangeEntry(comp *Competitor, rangeNum int, timeStr string) error {
	if rangeNum < 1 || rangeNum > rs.Config.FiringLines {
		return fmt.Errorf("неверный номер рубежа: %d", rangeNum)
	}
	rs.initShootingStats(comp)
	rs.logger.LogEvent(timeStr, fmt.Sprintf("The competitor(%d) is on the firing range(%d)", comp.ID, rangeNum))
	return nil
}

func (rs *RaceSystem) initShootingStats(comp *Competitor) {
	if len(comp.ShootingStats) <= rs.CurrentLap {
		comp.ShootingStats = append(comp.ShootingStats, make([][]bool, rs.Config.FiringLines))
	}
}

func (rs *RaceSystem) processShot(comp *Competitor, target int, timeStr string) error {
	if rs.CurrentLap >= len(comp.ShootingStats) {
		return fmt.Errorf("недопустимый круг: %d", rs.CurrentLap)
	}

	rangeNum := 0
	if rangeNum >= len(comp.ShootingStats[rs.CurrentLap]) {
		return fmt.Errorf("недопустимый рубеж: %d", rangeNum)
	}

	hit := determineHit()
	comp.ShootingStats[rs.CurrentLap][rangeNum] = append(
		comp.ShootingStats[rs.CurrentLap][rangeNum],
		hit,
	)

	rs.logger.LogEvent(timeStr, fmt.Sprintf("The target(%d) has been hit by competitor(%d)", comp.ID, target))

	if !hit {
		comp.PenaltyLaps++
	}
	return nil
}

func determineHit() bool {

	return true
}

func (rs *RaceSystem) processRangeExit(comp *Competitor, timeStr string) error {
	rangeNum := 0
	if rs.CurrentLap >= len(comp.ShootingStats) || rangeNum >= len(comp.ShootingStats[rs.CurrentLap]) {
		return errors.New("недопустимый рубеж")
	}

	misses := 5 - countHits(comp.ShootingStats[rs.CurrentLap][rangeNum])
	if misses > 0 {
		comp.PenaltyLaps += misses
		rs.logger.LogEvent(timeStr, fmt.Sprintf("The competitor(%d) left the firing range",
			comp.ID))
	}
	return nil
}

func countHits(shots []bool) int {
	count := 0
	for _, hit := range shots {
		if hit {
			count++
		}
	}
	return count
}

func (rs *RaceSystem) processPenaltyEntry(comp *Competitor, timeStr string) error {
	if comp.PenaltyLaps <= 0 {
		return errors.New("участник не имеет штрафных кругов")
	}
	rs.logger.LogEvent(timeStr, fmt.Sprintf("The competitor(%d) entered the penalty laps", comp.ID))
	return nil
}

func (rs *RaceSystem) processPenaltyExit(comp *Competitor, timeStr string) error {
	if comp.PenaltyLaps <= 0 {
		return errors.New("участник не имеет штрафных кругов")
	}
	comp.PenaltyLaps--
	rs.logger.LogEvent(timeStr, fmt.Sprintf("The competitor(%d) left the penalty laps", comp.ID))
	return nil
}

func (rs *RaceSystem) processLapCompletion(comp *Competitor, t time.Time, timeStr string) error {
	comp.LapsCompleted++
	if comp.LapsCompleted > rs.CurrentLap {
		rs.CurrentLap = comp.LapsCompleted
	}

	rs.logger.LogEvent(timeStr, fmt.Sprintf("The competitor(%d) ended the main lap", comp.ID))

	if comp.LapsCompleted >= rs.Config.Laps && comp.PenaltyLaps == 0 {
		return rs.finishCompetitor(comp, t, timeStr)
	}
	return nil
}

func (rs *RaceSystem) finishCompetitor(comp *Competitor, t time.Time, timeStr string) error {
	comp.Finished = true
	rs.generateOutgoingEvent(t, EventFinished, comp.ID, "", timeStr)
	return nil
}

func (rs *RaceSystem) disqualify(comp *Competitor, reason string, timeStr string) error {
	comp.Disqualified = true
	rs.generateOutgoingEvent(time.Now(), EventDisqualified, comp.ID, reason, timeStr)
	return nil
}

func (rs *RaceSystem) generateOutgoingEvent(eventTime time.Time, eventID int, competitorID int, message string, timeStr string) {
	event := fmt.Sprintf("[%s] %d %d", eventTime.Format("15:04:05.000"), eventID, competitorID)
	if message != "" {
		event += " " + message
	}
	rs.logger.LogEvent(timeStr, event)
}

func (rs *RaceSystem) getCompetitor(id int) *Competitor {
	if _, exists := rs.Competitors[id]; !exists {
		rs.Competitors[id] = &Competitor{ID: id}
	}
	return rs.Competitors[id]
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func (rs *RaceSystem) ProcessEventsFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "] ")
		if len(parts) < 2 {
			continue
		}

		timestampStr := parts[0][1:]
		data := parts[1]
		dataParts := strings.Fields(data)

		if len(dataParts) < 2 {
			continue
		}

		var eventID, competitorID int
		fmt.Sscanf(dataParts[0], "%d", &eventID)
		fmt.Sscanf(dataParts[1], "%d", &competitorID)

		params := ""
		if len(dataParts) > 2 {
			params = strings.Join(dataParts[2:], " ")
		}

		if err := rs.ProcessEvent(timestampStr, eventID, competitorID, params); err != nil {
			fmt.Printf("Error processing event: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	return nil
}
