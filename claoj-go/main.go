package main

import (
	"log"

	"github.com/CLAOJ/claoj-go/api"
	"github.com/CLAOJ/claoj-go/bridge"
	"github.com/CLAOJ/claoj-go/cache"
	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/events"
	"github.com/CLAOJ/claoj-go/jobs"
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

	// 7. Build router and run
	r := api.NewRouter()
	addr := ":" + config.C.Server.Port
	log.Printf("claoj-go HTTP API starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
