package store

import (
	"time"

	l4g "github.com/alecthomas/log4go"
	"github.com/jwilander/contributor-leaderboard/model"
)

type StoreResult struct {
	Data interface{}
	Err  error
}

type StoreChannel chan StoreResult

func Must(sc StoreChannel) interface{} {
	r := <-sc
	if r.Err != nil {
		l4g.Close()
		time.Sleep(time.Second)
		panic(r.Err)
	}

	return r.Data
}

type Store interface {
	Leaderboard() LeaderboardStore
	LeaderboardEntry() LeaderboardEntryStore
	Close()
	DropAllTables()
}

type LeaderboardStore interface {
	Save(leaderboard *model.Leaderboard) StoreChannel
	Get(id string) StoreChannel
	GetByName(name string) StoreChannel
}

type LeaderboardEntryStore interface {
	Save(entry *model.LeaderboardEntry) StoreChannel
	IncrementPoints(username string, leaderboardId string) StoreChannel
	GetRankings(leaderboardId string) StoreChannel
}
