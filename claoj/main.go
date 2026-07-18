package main

import (
	"log"

	"github.com/CLAOJ/claoj/api"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/bridge"
	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/events"
	"github.com/CLAOJ/claoj/jobs"
	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load configuration
	config.Load()

	// 2. Set Gin mode
	gin.SetMode(config.C.Server.Mode)

	// 3. Connect to dependencies
	db.Connect()
	cache.Connect()

	// 3b. Wire up refresh-token session storage. Redis-backed when
	// available so rotation-reuse detection survives restarts and works
	// across multiple API instances; falls back to an in-memory store
	// (single-process, resets on restart) if Redis isn't up.
	if cache.Client != nil {
		authHandlers.RefreshStore = tokenstore.NewRedisStore(cache.Client)
	} else {
		authHandlers.RefreshStore = tokenstore.NewMemoryStore()
		log.Printf("warning: refresh sessions in-memory; logins reset on restart")
	}

	// 3c. Wire up one-time token storage (password reset / email
	// verification) for the same reason: keep v2-only state out of the
	// shared MySQL schema.
	if cache.Client != nil {
		authHandlers.OneTimeTokens = tokenstore.NewRedisOneTime(cache.Client)
	} else {
		authHandlers.OneTimeTokens = tokenstore.NewMemoryOneTime()
		log.Printf("warning: one-time tokens in-memory; password-reset/email-verify links reset on restart")
	}

	// 4. Start the Judge Bridge TCP server
	judgeBridge := bridge.NewServer()
	go func() {
		if err := judgeBridge.Start(); err != nil {
			log.Fatalf("bridge: %v", err)
		}
	}()

	// 5. Initialize Asynq Background Tasks
	jobs.InitClient()
	jobs.SetBridge(judgeBridge)
	go jobs.StartWorker()

	// 6. Initialize Websocket Event Hub
	events.InitHub()

	// 7. Set bridge server reference for API handlers
	v2.SetBridgeServer(judgeBridge)

	// 8. Build router and run
	r := api.NewRouter()
	addr := ":" + config.C.Server.Port
	log.Printf("claoj-go HTTP API starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
