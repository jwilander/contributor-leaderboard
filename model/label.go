package model

import (
	"encoding/json"
	"io"
)

type Label struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func (l *Label) ToJson() string {
	b, err := json.Marshal(l)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func LabelFromJson(data io.Reader) *Label {
	decoder := json.NewDecoder(data)
	var o Label
	err := decoder.Decode(&o)
	if err == nil {
		return &o
	} else {
		return nil
	}
}
