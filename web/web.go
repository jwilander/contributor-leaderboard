package web

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"html/template"
	"io/ioutil"
	"net/http"

	l4g "github.com/alecthomas/log4go"
	"github.com/jwilander/contributor-leaderboard/model"
	"gopkg.in/fsnotify.v1"
)

var Templates *template.Template

type HtmlTemplatePage struct {
	TemplateName string
	Props        map[string]interface{}
}

func NewHtmlTemplatePage(templateName string, title string) *HtmlTemplatePage {
	props := make(map[string]interface{})
	props["Title"] = title
	return &HtmlTemplatePage{TemplateName: templateName, Props: props}
}

func (me *HtmlTemplatePage) Render(w http.ResponseWriter) {
	if err := Templates.ExecuteTemplate(w, me.TemplateName, me); err != nil {
		l4g.Error("Error rendering template, err=%v", err.Error())
	}
}

func InitWeb() {
	l4g.Debug("web.init.debug")

	mainrouter := Srv.Router

	l4g.Debug("Using client directory at %v", "web/static/")
	mainrouter.PathPrefix("/static/").Handler(staticHandler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/")))))

	mainrouter.HandleFunc("/", root).Methods("GET")
	mainrouter.HandleFunc("/event", handleEvent).Methods("POST")

	watchAndParseTemplates()
}

func watchAndParseTemplates() {
	l4g.Debug("Parsing templates at %v", "web/templates/")
	var err error
	if Templates, err = template.ParseGlob("web/templates/*.html"); err != nil {
		l4g.Error("Failed to parse templates %v", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		l4g.Error("Failed to create directory watcher %v", err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					l4g.Info("Re-parsing templates because of modified file %v", event.Name)
					if Templates, err = template.ParseGlob("web/templates/*.html"); err != nil {
						l4g.Error("Failed to parse templates %v", err)
					}
				}
			case err := <-watcher.Errors:
				l4g.Error("Failed in directory watcher %v", err)
			}
		}
	}()

	err = watcher.Add("web/templates/")
	if err != nil {
		l4g.Error("Failed to add directory to watcher %v", err)
	}
}

func staticHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=31556926, public")
		handler.ServeHTTP(w, r)
	})
}

func root(w http.ResponseWriter, r *http.Request) {
	page := NewHtmlTemplatePage("leaderboard", "Leaderboard")

	if result := <-Srv.Store.LeaderboardEntry().GetRankings(Srv.Leaderboard.Id); result.Err != nil {
		l4g.Error("Failed to load rankings, err=%v", result.Err.Error())
	} else {
		page.Props["Rankings"] = result.Data.([]*model.LeaderboardEntry)
	}

	w.Header().Set("Cache-Control", "no-cache, max-age=31556926, public")
	page.Render(w)
}

var exemptedUsers map[string]bool = map[string]bool{
	"jwilander":        true,
	"jasonblais":       true,
	"crspeller":        true,
	"coreyhulen":       true,
	"enahum":           true,
	"hmhealey":         true,
	"asaadmahmood":     true,
	"grundleborg":      true,
	"it33":             true,
	"lfbrock":          true,
	"esethna":          true,
	"hannaparks":       true,
	"mattermod":        true,
	"mattermost-build": true,
	"yang-chen1":       true,
	"yuya-oc":          true,
}

func handleEvent(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		l4g.Error("Unable to read request body, err=%v", err.Error())
		w.Write([]byte("fail"))
		return
	}

	if !CheckMAC(body, []byte(r.Header.Get("X-Hub-Signature")), []byte(*Srv.Cfg.WebhookToken)) {
		l4g.Error("Invalid HMAC signature")
		w.Write([]byte("fail"))
		return
	}

	event := model.EventFromJson(bytes.NewReader(body))

	w.Header().Set("Content-Type", "text/plain")

	if exemptedUsers[event.PullRequest.User.Login] {
		l4g.Debug("User %v exempted", event.PullRequest.User.Login)
		w.Write([]byte("ok"))
		return
	}

	if event.Action == "closed" && event.PullRequest.Merged {
		if result := <-Srv.Store.Label().Get(event.PullRequest.Id); result.Err != nil {
			l4g.Info("Unable to find label for this PR, did not count pull request err=%v", result.Err.Error())
			w.Write([]byte("ok"))
			return
		}

		entry := &model.LeaderboardEntry{
			LeaderboardId: Srv.Leaderboard.Id,
			Username:      event.PullRequest.User.Login,
		}

		if result := <-Srv.Store.LeaderboardEntry().Save(entry); result.Err != nil {
			l4g.Error("Unable to save entry, err=%v", result.Err.Error())
			w.Write([]byte("fail"))
			return
		}

		if result := <-Srv.Store.LeaderboardEntry().IncrementPoints(entry.Username, entry.LeaderboardId); result.Err != nil {
			l4g.Error("Unable to update points, err=%v", result.Err.Error())
			w.Write([]byte("fail"))
			return
		}

		go func() {
			if result := <-Srv.Store.Label().Delete(event.PullRequest.Id); result.Err != nil {
				l4g.Error("Unable to delete label, err=%v", result.Err.Error())
			}
		}()

		l4g.Debug("Pull request with id %v merged", event.PullRequest.Id)
	} else if event.Action == "labeled" && event.Label.Name == "Hackfest" {

		label := &model.Label{
			Id:   event.PullRequest.Id,
			Name: event.Label.Name,
		}

		if result := <-Srv.Store.Label().Save(label); result.Err != nil {
			l4g.Error("Unable to save label, err=%v", result.Err.Error())
			w.Write([]byte("fail"))
			return
		}
		l4g.Debug("Label added for pull request id %v", event.PullRequest.Id)
	} else if event.Action == "unlabeled" && event.Label.Name == "Hackfest" {
		if result := <-Srv.Store.Label().Delete(event.PullRequest.Id); result.Err != nil {
			l4g.Error("Unable to delete label, err=%v", result.Err.Error())
			w.Write([]byte("fail"))
			return
		}
		l4g.Debug("Label removed for pull request id %v", event.PullRequest.Id)
	} else {
		l4g.Debug("Unrecognized action '%v'", event.Action)
	}

	w.Write([]byte("ok"))
}

func CheckMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	expectedSig := "sha1=" + hex.EncodeToString(expectedMAC)

	return hmac.Equal(messageMAC, []byte(expectedSig))
}
