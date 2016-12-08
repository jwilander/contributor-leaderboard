package store

import (
	dbsql "database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	l4g "github.com/alecthomas/log4go"

	"github.com/go-gorp/gorp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jwilander/contributor-leaderboard/model"
	_ "github.com/lib/pq"
)

const (
	INDEX_TYPE_FULL_TEXT = "full_text"
	INDEX_TYPE_DEFAULT   = "default"
)

const (
	EXIT_CREATE_TABLE                = 100
	EXIT_DB_OPEN                     = 101
	EXIT_PING                        = 102
	EXIT_NO_DRIVER                   = 103
	EXIT_TABLE_EXISTS                = 104
	EXIT_TABLE_EXISTS_MYSQL          = 105
	EXIT_COLUMN_EXISTS               = 106
	EXIT_DOES_COLUMN_EXISTS_POSTGRES = 107
	EXIT_DOES_COLUMN_EXISTS_MYSQL    = 108
	EXIT_DOES_COLUMN_EXISTS_MISSING  = 109
	EXIT_CREATE_COLUMN_POSTGRES      = 110
	EXIT_CREATE_COLUMN_MYSQL         = 111
	EXIT_CREATE_COLUMN_MISSING       = 112
	EXIT_REMOVE_COLUMN               = 113
	EXIT_RENAME_COLUMN               = 114
	EXIT_MAX_COLUMN                  = 115
	EXIT_ALTER_COLUMN                = 116
	EXIT_CREATE_INDEX_POSTGRES       = 117
	EXIT_CREATE_INDEX_MYSQL          = 118
	EXIT_CREATE_INDEX_FULL_MYSQL     = 119
	EXIT_CREATE_INDEX_MISSING        = 120
	EXIT_REMOVE_INDEX_POSTGRES       = 121
	EXIT_REMOVE_INDEX_MYSQL          = 122
	EXIT_REMOVE_INDEX_MISSING        = 123
)

type SqlStore struct {
	master           *gorp.DbMap
	leaderboard      LeaderboardStore
	leaderboardEntry LeaderboardEntryStore
	label            LabelStore
}

func initConnection(connUrl string) *SqlStore {
	sqlStore := &SqlStore{}

	sqlStore.master = setupConnection("master", connUrl)

	return sqlStore
}

func NewSqlStore(connUrl string) Store {

	sqlStore := initConnection(connUrl)

	sqlStore.leaderboard = NewSqlLeaderboardStore(sqlStore)
	sqlStore.leaderboardEntry = NewSqlLeaderboardEntryStore(sqlStore)
	sqlStore.label = NewSqlLabelStore(sqlStore)

	err := sqlStore.master.CreateTablesIfNotExists()
	if err != nil {
		l4g.Critical("Unable to create database tables", err)
		time.Sleep(time.Second)
		os.Exit(EXIT_CREATE_TABLE)
	}

	sqlStore.leaderboard.(*SqlLeaderboardStore).CreateIndexesIfNotExists()
	sqlStore.leaderboardEntry.(*SqlLeaderboardEntryStore).CreateIndexesIfNotExists()
	sqlStore.label.(*SqlLabelStore).CreateIndexesIfNotExists()

	return sqlStore
}

func setupConnection(con_type string, dataSource string) *gorp.DbMap {

	db, err := dbsql.Open("postgres", dataSource)
	if err != nil {
		l4g.Critical("Unable to open connection to database, err=%v", err)
		time.Sleep(time.Second)
		os.Exit(EXIT_DB_OPEN)
	}

	l4g.Info("Pinging database")
	err = db.Ping()
	if err != nil {
		l4g.Critical("Unable to ping database, err=%v", err)
		time.Sleep(time.Second)
		os.Exit(EXIT_PING)
	}

	dbmap := &gorp.DbMap{Db: db, TypeConverter: mattermConverter{}, Dialect: gorp.PostgresDialect{}}

	return dbmap
}

func (ss *SqlStore) DoesTableExist(tableName string) bool {
	count, err := ss.GetMaster().SelectInt(
		`SELECT count(relname) FROM pg_class WHERE relname=$1`,
		strings.ToLower(tableName),
	)

	if err != nil {
		l4g.Critical("Errored checking if table exists", err)
		time.Sleep(time.Second)
		os.Exit(EXIT_TABLE_EXISTS)
	}

	return count > 0
}

func (ss *SqlStore) DoesColumnExist(tableName string, columnName string) bool {
	count, err := ss.GetMaster().SelectInt(
		`SELECT COUNT(0)
			FROM   pg_attribute
			WHERE  attrelid = $1::regclass
			AND    attname = $2
			AND    NOT attisdropped`,
		strings.ToLower(tableName),
		strings.ToLower(columnName),
	)

	if err != nil {
		if err.Error() == "pq: relation \""+strings.ToLower(tableName)+"\" does not exist" {
			return false
		}

		l4g.Critical("Errored checking if column exists", err)
		time.Sleep(time.Second)
		os.Exit(EXIT_DOES_COLUMN_EXISTS_POSTGRES)
	}

	return count > 0
}

func (ss *SqlStore) CreateColumnIfNotExists(tableName string, columnName string, mySqlColType string, postgresColType string, defaultValue string) bool {

	if ss.DoesColumnExist(tableName, columnName) {
		return false
	}

	_, err := ss.GetMaster().Exec("ALTER TABLE " + tableName + " ADD " + columnName + " " + postgresColType + " DEFAULT '" + defaultValue + "'")
	if err != nil {
		l4g.Critical("Erroed creating column", err)
		time.Sleep(time.Second)
		os.Exit(EXIT_CREATE_COLUMN_POSTGRES)
	}

	return true

}

func (ss *SqlStore) RemoveColumnIfExists(tableName string, columnName string) bool {

	if !ss.DoesColumnExist(tableName, columnName) {
		return false
	}

	_, err := ss.GetMaster().Exec("ALTER TABLE " + tableName + " DROP COLUMN " + columnName)
	if err != nil {
		l4g.Critical("Errored removing column", err)
		time.Sleep(time.Second)
		os.Exit(EXIT_REMOVE_COLUMN)
	}

	return true
}

func (ss *SqlStore) CreateUniqueIndexIfNotExists(indexName string, tableName string, columnName string) {
	ss.createIndexIfNotExists(indexName, tableName, columnName, INDEX_TYPE_DEFAULT, true)
}

func (ss *SqlStore) CreateIndexIfNotExists(indexName string, tableName string, columnName string) {
	ss.createIndexIfNotExists(indexName, tableName, columnName, INDEX_TYPE_DEFAULT, false)
}

func (ss *SqlStore) CreateFullTextIndexIfNotExists(indexName string, tableName string, columnName string) {
	ss.createIndexIfNotExists(indexName, tableName, columnName, INDEX_TYPE_FULL_TEXT, false)
}

func (ss *SqlStore) createIndexIfNotExists(indexName string, tableName string, columnName string, indexType string, unique bool) {

	uniqueStr := ""
	if unique {
		uniqueStr = "UNIQUE "
	}

	_, err := ss.GetMaster().SelectStr("SELECT $1::regclass", indexName)
	// It should fail if the index does not exist
	if err == nil {
		return
	}

	query := ""
	if indexType == INDEX_TYPE_FULL_TEXT {
		postgresColumnNames := convertMySQLFullTextColumnsToPostgres(columnName)
		query = "CREATE INDEX " + indexName + " ON " + tableName + " USING gin(to_tsvector('english', " + postgresColumnNames + "))"
	} else {
		query = "CREATE " + uniqueStr + "INDEX " + indexName + " ON " + tableName + " (" + columnName + ")"
	}

	_, err = ss.GetMaster().Exec(query)
	if err != nil {
		l4g.Critical("Errored creating index", err)
		time.Sleep(time.Second)
		os.Exit(EXIT_CREATE_INDEX_POSTGRES)
	}
}

func IsUniqueConstraintError(err string, indexName []string) bool {
	unique := strings.Contains(err, "unique constraint") || strings.Contains(err, "Duplicate entry")
	field := false
	for _, contain := range indexName {
		if strings.Contains(err, contain) {
			field = true
			break
		}
	}

	return unique && field
}

func (ss *SqlStore) GetMaster() *gorp.DbMap {
	return ss.master
}

func (ss *SqlStore) Close() {
	l4g.Info("Closing database conections")
	ss.master.Db.Close()
}

func (ss *SqlStore) Leaderboard() LeaderboardStore {
	return ss.leaderboard
}

func (ss *SqlStore) LeaderboardEntry() LeaderboardEntryStore {
	return ss.leaderboardEntry
}

func (ss *SqlStore) Label() LabelStore {
	return ss.label
}

func (ss *SqlStore) DropAllTables() {
	ss.master.TruncateTables()
}

type mattermConverter struct{}

func (me mattermConverter) ToDb(val interface{}) (interface{}, error) {

	switch t := val.(type) {
	case model.StringMap:
		return model.MapToJson(t), nil
	case model.StringArray:
		return model.ArrayToJson(t), nil
	case model.StringInterface:
		return model.StringInterfaceToJson(t), nil
	}

	return val, nil
}

func (me mattermConverter) FromDb(target interface{}) (gorp.CustomScanner, bool) {
	switch target.(type) {
	case *model.StringMap:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return errors.New("store.sql.convert_string_map")
			}
			b := []byte(*s)
			return json.Unmarshal(b, target)
		}
		return gorp.CustomScanner{new(string), target, binder}, true
	case *model.StringArray:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return errors.New("store.sql.convert_string_array")
			}
			b := []byte(*s)
			return json.Unmarshal(b, target)
		}
		return gorp.CustomScanner{new(string), target, binder}, true
	case *model.StringInterface:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return errors.New("store.sql.convert_string_interface")
			}
			b := []byte(*s)
			return json.Unmarshal(b, target)
		}
		return gorp.CustomScanner{new(string), target, binder}, true
	}

	return gorp.CustomScanner{}, false
}

func convertMySQLFullTextColumnsToPostgres(columnNames string) string {
	columns := strings.Split(columnNames, ", ")
	concatenatedColumnNames := ""
	for i, c := range columns {
		concatenatedColumnNames += c
		if i < len(columns)-1 {
			concatenatedColumnNames += " || ' ' || "
		}
	}

	return concatenatedColumnNames
}
