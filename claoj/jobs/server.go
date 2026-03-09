package jobs

import (
	"log"

	"github.com/CLAOJ/claoj/config"
	"github.com/hibiken/asynq"
)

var Client *asynq.Client

// InitClient creates the Asynq client for enqueueing tasks.
// Must be called after config.Load().
func InitClient() {
	redisOpt := asynq.RedisClientOpt{
		Addr:     config.C.Redis.Addr,
		Password: config.C.Redis.Password,
		DB:       config.C.Redis.DB,
	}
	Client = asynq.NewClient(redisOpt)
	log.Println("jobs: Asynq client initialized")
}

// StartWorker spins up the background worker that processes all Asynq queues.
func StartWorker() {
	redisOpt := asynq.RedisClientOpt{
		Addr:     config.C.Redis.Addr,
		Password: config.C.Redis.Password,
		DB:       config.C.Redis.DB,
	}

	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			// Concurrency settings could be pulled from config.
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	mux := asynq.NewServeMux()

	// Register all task handlers
	mux.HandleFunc(TypeJudgeSubmission, HandleJudgeSubmissionTask)
	mux.HandleFunc(TypeRescoreProblem, HandleRescoreTask)
	mux.HandleFunc(TypeRateContest, HandleRateContestTask)
	mux.HandleFunc(TypeUserExport, HandleUserExportTask)

	log.Println("jobs: Starting Asynq background worker")
	if err := srv.Run(mux); err != nil {
		log.Fatalf("jobs: could not start worker server: %v", err)
	}
}
