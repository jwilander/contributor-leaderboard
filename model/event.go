package model

import (
	"encoding/json"
	"io"
)

type Event struct {
	Action      string           `json:"action"`
	PullRequest EventPullRequest `json:"pull_request"`
}

type EventPullRequest struct {
	Merged bool      `json:"merged"`
	User   EventUser `json:"user"`
}

type EventUser struct {
	Id    int    `json:"id"`
	Login string `json:"login"`
}

func (l *Event) ToJson() string {
	b, err := json.Marshal(l)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func EventFromJson(data io.Reader) *Event {
	decoder := json.NewDecoder(data)
	var o Event
	err := decoder.Decode(&o)
	if err == nil {
		return &o
	} else {
		return nil
	}
}
