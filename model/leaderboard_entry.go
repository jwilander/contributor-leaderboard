package model

import (
	"encoding/json"
)

type LeaderboardEntry struct {
	LeaderboardId string `json:"leaderboard_id"`
	Username      string `json:"username"`
	Points        int    `json:"points"`
}

func (l *LeaderboardEntry) PreSave() {
	l.Points = 0
}

func (l *LeaderboardEntry) ToJson() string {
	b, err := json.Marshal(l)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}
