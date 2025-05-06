package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Laps        int    `json:"laps"`
	LapLen      int    `json:"lapLen"`
	PenaltyLen  int    `json:"penaltyLen"`
	FiringLines int    `json:"firingLines"`
	Start       string `json:"start"`
	StartDelta  string `json:"startDelta"`
}

func LoadConfig(path string) (*Config, error) {
	var config Config
	data, err := os.ReadFile(path)
	if err != nil {
		return &config, err
	}
	err = json.Unmarshal(data, &config)
	return &config, err
}
