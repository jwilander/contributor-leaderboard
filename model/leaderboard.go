package model

import (
	"encoding/json"
)

type Leaderboard struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (l *Leaderboard) PreSave() {
	if l.Id == "" {
		l.Id = NewId()
	}
}

func (l *Leaderboard) ToJson() string {
	b, err := json.Marshal(l)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func LeaderboardFromJson(data string) (*Leaderboard, error) {
	var leaderboard Leaderboard
	if err := json.Unmarshal([]byte(data), leaderboard); err == nil {
		return &leaderboard, nil
	} else {
		return nil, err
	}
}
