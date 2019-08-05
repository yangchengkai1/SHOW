package controller

import (
	"log"
	"net/http"

	"gopkg.in/go-playground/webhooks.v5/github"

	r "github.com/dancannon/gorethink"
	"github.com/gin-gonic/gin"
	"github.com/yangchengkai1/webhooks/model"
	yuque "github.com/yangchengkai1/webhooks/model"
)

var yqhook struct {
	Data yuque.DocDetailSerializer `json:"data"`
}

// Session -
type Session struct {
	ys *r.Session
	gs *r.Session
}

// RegisterRouter -
func RegisterRouter(router gin.IRouter) {
	var ss *Session

	session, err := r.Connect(r.ConnectOpts{
		Address: "localhost",
	})
	if err != nil {
		log.Fatal(err)
	}

	ys, err := model.Create("yuque", "yuque", session)
	if err != nil {
		log.Fatal(err)
	}

	gs, err := model.Create("github", "github", session)
	if err != nil {
		log.Fatal(err)
	}

	ss = &Session{ys: ys, gs: gs}

	router.POST("/github/webhook", ss.githubStore)
	router.POST("/yuque/webhook", ss.yuqueStore)
	router.POST("/select", ss.selectHandler)
}

func (s Session) githubStore(c *gin.Context) {
	hook, _ := github.New(github.Options.Secret("MyGitHubSuperSecret"))

	payload, err := hook.Parse(c.Request, github.PushEvent, github.PullRequestEvent)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest})
		return
	}

	switch payload.(type) {
	case github.PushPayload:
		push := payload.(github.PushPayload)
		err = model.InsertGitRecord(push, s.gs)
		if err != nil {
			log.Println("0")
			c.Error(err)
			c.JSON(http.StatusMethodNotAllowed, gin.H{"status": http.StatusMethodNotAllowed})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}

func (s Session) yuqueStore(c *gin.Context) {
	err := c.ShouldBind(&yqhook)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest})
		return
	}

	err = model.InsertYuQueRecord(yqhook.Data.Body, yqhook.Data.ActionType, yqhook.Data.UpdatedAt, yqhook.Data.User.Name, s.ys)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusMethodNotAllowed, gin.H{"status": http.StatusMethodNotAllowed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}

func (s Session) selectHandler(c *gin.Context) {
	var (
		term struct {
			DBName    string `json:"db_name"    binding:"required"`
			TableName string `json:"table_name" binding:"required"`
			Field     string `json:"field"      binding:"required"`
			Value     string `json:"value"      binding:"required"`
		}

		session *r.Session
	)

	err := c.ShouldBind(&term)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest})
		return
	}

	switch term.TableName {
	case "yuque":
		session = s.ys
	case "github":
		session = s.gs
	}

	all, err := model.SelectRecord(session, term.DBName, term.TableName, term.Field, term.Value)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusMethodNotAllowed, gin.H{"status": http.StatusMethodNotAllowed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"all": all})
}
