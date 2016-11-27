package model

import (
	"testing"
)

func TestLeaderboard(t *testing.T) {
	l := &Leaderboard{Id: "", Name: "TestLeaderboard"}
	l.PreSave()

	if l.Id == "" {
		t.Fatal("id should be set")
	}

	str := l.ToJson()
	l2 := LeaderboardFromJson(str)

	if l2.Id != l.Id {
		t.Fatal("ids should match")
	}

	if l2.Name != l.Name {
		t.Fatal("names should match")
	}
}
