package main

import (
	"os"
	"os/signal"
	"syscall"

	l4g "github.com/alecthomas/log4go"
	"github.com/jwilander/contributor-leaderboard/model"
	"github.com/jwilander/contributor-leaderboard/web"
)

func main() {
	l4g.Info("Starting up leaderboard server")

	config := model.Config{}

	databaseSource := os.Getenv("DATABASE_URL")
	if len(databaseSource) == 0 {
		databaseSource = "postgres://mmuser:mostest@dockerhost:5432/leaderboard?sslmode=disable&connect_timeout=10"
	}

	config.DatabaseSource = new(string)
	*config.DatabaseSource = databaseSource
	config.LeaderboardName = new(string)
	*config.LeaderboardName = "Holiday Hackfest Leaderboard"
	config.WebhookToken = new(string)
	*config.WebhookToken = os.Getenv("WEBHOOK_TOKEN")

	web.StartServer(config)

	// wait for kill signal before attempting to gracefully shutdown
	// the running service
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-c

	l4g.Info("Stopping leaderboard server")
	web.StopServer()
}
