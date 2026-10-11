// Command claoj-judge pulls submissions from the asynq "judge" queue,
// grades them on a local tiervnoj (DMOJ) process and POSTs every judge
// packet to the backend's /internal/judge/event.
package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
)

func main() {
	redisAddr := mustEnv("REDIS_ADDR")
	backendURL := mustEnv("BACKEND_URL")
	name := mustEnv("JUDGE_NAME")
	key := mustEnv("JUDGE_KEY")
	redisDB, err := strconv.Atoi(env("REDIS_DB", "0"))
	if err != nil {
		log.Fatalf("REDIS_DB: %v", err)
	}
	port := env("CHILD_PORT", "9999")

	// 1. Loopback bridge for tiervnoj. The forwarder is wired in before
	// tiervnoj spawns: every authenticated handshake reports the executor
	// list as a "judge-runtimes" event, synchronously ahead of Ready(), so
	// the first report always beats the queue's first task — and every
	// reconnect reports it again, which is what keeps the backend's
	// delete-then-insert honest after a reconnect.
	fwd := &Forwarder{
		URL:     strings.TrimRight(backendURL, "/") + "/internal/judge/event",
		Name:    name,
		Key:     key,
		Client:  &http.Client{Timeout: 30 * time.Second},
		MaxWait: 5 * time.Minute,
		Backoff: 500 * time.Millisecond,
	}
	child := NewChild(key)
	child.onRuntimes = func(p Packet) {
		if err := fwd.Post(p); err != nil {
			log.Printf("judge-runtimes: %v", err)
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	go child.Serve(ln)
	go child.KeepAlive(60 * time.Second)

	// 2. Spawn tiervnoj; if it dies, die with it so the orchestrator restarts us.
	cmd := exec.Command(env("TIERVNOJ_ENTRY", "/judge/.docker/entry"), "run",
		"-p", port, "-c", env("JUDGE_YML", "/problems/judge.yml"), "--no-watchdog",
		"127.0.0.1", name, key)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("start tiervnoj: %v", err)
	}
	go func() { log.Fatalf("tiervnoj exited: %v", cmd.Wait()) }()

	// tiervnoj self-tests every executor before connecting; that takes a while.
	for deadline := time.Now().Add(15 * time.Minute); !child.Ready(); time.Sleep(time.Second) {
		if time.Now().After(deadline) {
			log.Fatal("tiervnoj did not connect within 15 minutes")
		}
	}
	log.Printf("judge %q ready", name)

	// 3. Pull work.
	agent := &Agent{child: child, abortWait: 30 * time.Second}
	agent.post = func(p Packet) error {
		err := fwd.Post(p)
		if errors.Is(err, errUnauthorized) {
			log.Fatalf("backend rejected judge %q: JUDGE_NAME/JUDGE_KEY must match a judge_judge row", name)
		}
		return err
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: os.Getenv("REDIS_PASSWORD"), DB: redisDB},
		asynq.Config{
			Concurrency:     1, // one tiervnoj grades one submission at a time; scale with replicas
			Queues:          map[string]int{"judge": 1},
			ShutdownTimeout: 25 * time.Second, // keep the container stop grace period above this
			RetryDelayFunc:  func(int, error, *asynq.Task) time.Duration { return time.Second },
		},
	)
	if err := srv.Start(agent); err != nil {
		log.Fatalf("asynq: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
	<-sig
	agent.stopping.Store(true)
	srv.Shutdown()
	os.Exit(0)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("%s is required", k)
	}
	return v
}
