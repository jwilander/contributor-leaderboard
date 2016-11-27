package web

import (
	"net/http"
	"time"

	l4g "github.com/alecthomas/log4go"
	"github.com/gorilla/mux"
	"github.com/jwilander/contributor-leaderboard/model"
	"github.com/jwilander/contributor-leaderboard/store"
)

type Server struct {
	Store       store.Store
	Router      *mux.Router
	Server      *http.Server
	Cfg         model.Config
	Leaderboard *model.Leaderboard
}

type CorsWrapper struct {
	router *mux.Router
}

var Srv *Server

func StartServer(config model.Config) {
	Srv = &Server{}

	Srv.Cfg = config

	Srv.Store = store.NewSqlStore(*config.DatabaseSource)

	Srv.Router = mux.NewRouter()

	Srv.Server = &http.Server{
		Addr:         ":8075",
		Handler:      Srv.Router,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	leaderboard := &model.Leaderboard{
		Name: *config.LeaderboardName,
	}

	if result := <-Srv.Store.Leaderboard().Save(leaderboard); result.Err != nil {
		l4g.Critical("Unable to create leaderboard, err=%v", result.Err.Error())
		return
	} else {
		Srv.Leaderboard = result.Data.(*model.Leaderboard)
	}

	InitWeb()

	go func() {
		Srv.Server.ListenAndServe()
	}()
}

func StopServer() {
	Srv.Store.Close()
}
