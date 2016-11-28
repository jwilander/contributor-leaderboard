package store

import (
	"errors"

	"github.com/jwilander/contributor-leaderboard/model"
)

type SqlLeaderboardStore struct {
	*SqlStore
}

func NewSqlLeaderboardStore(sqlStore *SqlStore) LeaderboardStore {
	ls := &SqlLeaderboardStore{sqlStore}

	db := sqlStore.GetMaster()
	table := db.AddTableWithName(model.Leaderboard{}, "Leaderboards").SetKeys(false, "Id")
	table.ColMap("Id").SetMaxSize(26)
	table.ColMap("Name").SetMaxSize(64).SetUnique(true)

	return ls
}

func (ls SqlLeaderboardStore) CreateIndexesIfNotExists() {
}

func (ls SqlLeaderboardStore) Save(leaderboard *model.Leaderboard) StoreChannel {

	storeChannel := make(StoreChannel, 1)

	go func() {
		result := StoreResult{}

		if len(leaderboard.Id) > 0 {
			result.Err = errors.New("Cannot save existing leaderboard, leaderboard_id=" + leaderboard.Id)
			storeChannel <- result
			close(storeChannel)
			return
		}

		existing := model.Leaderboard{}

		if err := ls.GetMaster().SelectOne(&existing, "SELECT * FROM Leaderboards WHERE Name = :Name", map[string]interface{}{"Name": leaderboard.Name}); err != nil {
			leaderboard.PreSave()

			if err := ls.GetMaster().Insert(leaderboard); err != nil {
				result.Err = errors.New("Error saving leaderboard, leaderboard_id=" + leaderboard.Id + ", " + err.Error())
			} else {
				result.Data = leaderboard
			}
		} else {
			result.Data = &existing
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ls SqlLeaderboardStore) Get(id string) StoreChannel {

	storeChannel := make(StoreChannel, 1)

	go func() {
		result := StoreResult{}

		if obj, err := ls.GetMaster().Get(model.Leaderboard{}, id); err != nil {
			result.Err = errors.New("Error getting leaderboard, leaderboard_id=" + id + ", " + err.Error())
		} else if obj == nil {
			result.Err = errors.New("Missing leaderboard, leaderboard_id=" + id)
		} else {
			result.Data = obj.(*model.Leaderboard)
		}

		storeChannel <- result
		close(storeChannel)

	}()

	return storeChannel
}

func (ls SqlLeaderboardStore) GetByName(name string) StoreChannel {

	storeChannel := make(StoreChannel, 1)

	go func() {
		result := StoreResult{}

		leaderboard := model.Leaderboard{}

		if err := ls.GetMaster().SelectOne(&leaderboard, "SELECT * FROM Leaderboards WHERE Leaderboardname = :Name", map[string]interface{}{"Name": name}); err != nil {
			result.Err = errors.New("Error getting leaderboard by name, name=" + name + ", " + err.Error())
		}

		result.Data = &leaderboard

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}
