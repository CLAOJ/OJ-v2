package main

import (
	"context"
	"log"
	"time"

	"github.com/CLAOJ/claoj/api"
	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auditlog"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/events"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()
	gin.SetMode(config.C.Server.Mode)

	db.Connect()
	cache.Connect() // exits if Redis is unreachable

	// Session, one-time-token and audit-log state live in Redis so every
	// replica shares them and the MySQL schema stays Django-owned.
	authHandlers.RefreshStore = tokenstore.NewRedisStore(cache.Client)
	authHandlers.OneTimeTokens = tokenstore.NewRedisOneTime(cache.Client)
	auditlog.Default = auditlog.NewRedisStore(cache.Client)

	// Background jobs. Submissions go to asynq queue "judge", consumed only
	// by claoj-judge agents, which POST results to /internal/judge/event.
	jobs.InitClient()
	go jobs.StartWorker()
	if config.C.Judge.Name != "" {
		go jobs.RunJudgeStatusLoop(context.Background(), config.C.Judge.Name, 30*time.Second)
	} else {
		log.Printf("warning: JUDGE_NAME unset; judge online status is not synced")
	}

	events.InitHub()
	v2.SetBridgeServer(jobs.JudgeAborter{})

	r := api.NewRouter()
	addr := ":" + config.C.Server.Port
	log.Printf("claoj-go HTTP API starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
