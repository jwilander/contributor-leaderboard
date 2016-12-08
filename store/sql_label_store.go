package store

import (
	"errors"
	"fmt"

	"github.com/jwilander/contributor-leaderboard/model"
)

type SqlLabelStore struct {
	*SqlStore
}

func NewSqlLabelStore(sqlStore *SqlStore) LabelStore {
	ls := &SqlLabelStore{sqlStore}

	db := sqlStore.GetMaster()
	table := db.AddTableWithName(model.Label{}, "Label")
	table.ColMap("Name").SetMaxSize(256)

	return ls
}

func (ls SqlLabelStore) CreateIndexesIfNotExists() {
}

func (ls SqlLabelStore) Save(label *model.Label) StoreChannel {

	storeChannel := make(StoreChannel, 1)

	go func() {
		result := StoreResult{}

		if label.Id == 0 {
			result.Err = errors.New(fmt.Sprintf("Bad id, id=%v", label.Id))
			storeChannel <- result
			close(storeChannel)
			return
		}

		if err := ls.GetMaster().Insert(label); err != nil {
			result.Err = errors.New(fmt.Sprintf("Error saving label, id=%v, err=%v", label.Id, err.Error()))
		} else {
			result.Data = label
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ls SqlLabelStore) Get(labelId int) StoreChannel {

	storeChannel := make(StoreChannel, 1)

	go func() {
		result := StoreResult{}

		label := model.Label{}

		if err := ls.GetMaster().SelectOne(&label, "SELECT * FROM Label WHERE Id = :Id", map[string]interface{}{"Id": labelId}); err != nil {
			result.Err = errors.New(fmt.Sprintf("Error getting label, id=%v, err=%v", labelId, err.Error()))
		} else {
			result.Data = &label
		}

		storeChannel <- result
		close(storeChannel)

	}()

	return storeChannel
}

func (ls SqlLabelStore) Delete(labelId int) StoreChannel {

	storeChannel := make(StoreChannel, 1)

	go func() {
		result := StoreResult{}

		if _, err := ls.GetMaster().Exec("DELETE FROM Label WHERE Id = :Id", map[string]interface{}{"Id": labelId}); err != nil {
			result.Err = errors.New(fmt.Sprintf("Error deleting label, id=%v, err=%v", labelId, err.Error()))
		}

		storeChannel <- result
		close(storeChannel)

	}()

	return storeChannel
}
