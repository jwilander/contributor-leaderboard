package store

import (
	"errors"

	"github.com/jwilander/contributor-leaderboard/model"
)

type SqlLeaderboardEntryStore struct {
	*SqlStore
}

func NewSqlLeaderboardEntryStore(sqlStore *SqlStore) LeaderboardEntryStore {
	ls := &SqlLeaderboardEntryStore{sqlStore}

	db := sqlStore.GetMaster()
	table := db.AddTableWithName(model.LeaderboardEntry{}, "LeaderboardEntry")
	table.ColMap("LeaderboardId").SetMaxSize(26)
	table.ColMap("Username").SetMaxSize(128).SetUnique(true)

	return ls
}

func (ls SqlLeaderboardEntryStore) CreateIndexesIfNotExists() {
}

func (ls SqlLeaderboardEntryStore) Save(entry *model.LeaderboardEntry) StoreChannel {

	storeChannel := make(StoreChannel, 1)

	go func() {
		result := StoreResult{}

		if len(entry.LeaderboardId) != 26 {
			result.Err = errors.New("Bad leaderboard_id, leaderboard_id=" + entry.LeaderboardId)
			storeChannel <- result
			close(storeChannel)
			return
		}

		existing := model.LeaderboardEntry{}

		if err := ls.GetMaster().SelectOne(&existing, "SELECT * FROM LeaderboardEntry WHERE Username = :Username", map[string]interface{}{"Username": entry.Username}); err != nil {
			entry.PreSave()

			if err := ls.GetMaster().Insert(entry); err != nil {
				result.Err = errors.New("Error saving leaderboard entry, username=" + entry.Username + ", " + err.Error())
			} else {
				result.Data = entry
			}
		} else {
			result.Data = existing
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ls SqlLeaderboardEntryStore) IncrementPoints(username string, leaderboardId string) StoreChannel {

	storeChannel := make(StoreChannel, 1)

	go func() {
		result := StoreResult{}

		if _, err := ls.GetMaster().Exec("UPDATE LeaderboardEntry SET Points = Points + 1 WHERE Username = :Username AND LeaderboardId = :Id", map[string]interface{}{"Username": username, "Id": leaderboardId}); err != nil {
			result.Err = errors.New("Error incrementing points, leaderboard_id=" + leaderboardId + ", " + err.Error())
		}

		storeChannel <- result
		close(storeChannel)

	}()

	return storeChannel
}
