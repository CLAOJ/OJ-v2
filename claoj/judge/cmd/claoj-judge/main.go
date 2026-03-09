package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/CLAOJ/claoj/judge/core"
	"github.com/CLAOJ/claoj/judge/protocol"
)

func main() {
	// Command-line flags
	configPath := flag.String("c", "~/.claojrc", "Configuration file path")
	serverHost := flag.String("server", "", "Judge server host")
	serverPort := flag.Int("port", 9999, "Judge server port")
	judgeName := flag.String("name", "", "Judge name")
	judgeKey := flag.String("key", "", "Judge authentication key")
	apiHost := flag.String("api-host", "127.0.0.1", "API listen address")
	apiPort := flag.Int("api-port", 9998, "API listen port")
	logFile := flag.String("log", "", "Log file path")
	noWatchdog := flag.Bool("no-watchdog", false, "Disable problem directory watcher")
	skipSelfTest := flag.Bool("skip-self-test", false, "Skip executor self-tests")

	flag.Parse()

	// Validate required arguments
	if *serverHost == "" {
		log.Fatal("Server host is required")
	}
	if *judgeName == "" || *judgeKey == "" {
		log.Fatal("Judge name and key are required")
	}

	// Load configuration
	cfg, err := core.LoadConfig(*configPath)
	if err != nil {
		log.Printf("Warning: Could not load config file: %v", err)
		cfg = core.DefaultConfig()
	}

	// Override config with command-line flags
	cfg.ServerHost = *serverHost
	cfg.ServerPort = *serverPort
	if *judgeName != "" {
		cfg.JudgeName = *judgeName
	}
	if *judgeKey != "" {
		cfg.JudgeKey = *judgeKey
	}
	cfg.APIHost = *apiHost
	cfg.APIPort = *apiPort
	if *logFile != "" {
		cfg.LogFile = *logFile
	}
	cfg.NoWatchdog = *noWatchdog
	cfg.SkipSelfTest = *skipSelfTest

	// Setup logging
	if err := setupLogging(cfg.LogFile); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	log.Printf("Starting CLAOJ Judge v2.0")
	log.Printf("Connecting to %s:%d as %s", cfg.ServerHost, cfg.ServerPort, cfg.JudgeName)

	// Create judge instance
	judge, err := core.NewJudge(cfg)
	if err != nil {
		log.Fatalf("Failed to create judge: %v", err)
	}

	// Connect to server
	packetManager, err := protocol.NewPacketManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create packet manager: %v", err)
	}
	judge.SetPacketManager(packetManager)

	// Start API server (for local testing)
	if cfg.APIPort > 0 {
		go startAPIServer(cfg.APIHost, cfg.APIPort, judge)
		log.Printf("API server listening on %s:%d", cfg.APIHost, cfg.APIPort)
	}

	// Start problem watcher
	if !cfg.NoWatchdog {
		if err := startProblemWatcher(judge); err != nil {
			log.Printf("Warning: Failed to start problem watcher: %v", err)
		}
	}

	// Run executor self-tests
	if !cfg.SkipSelfTest {
		log.Println("Running executor self-tests...")
		if err := runSelfTests(judge); err != nil {
			log.Printf("Warning: Self-test failed: %v", err)
		}
	}

	// Start listening for submissions
	log.Println("Waiting for submissions...")
	if err := judge.Listen(); err != nil {
		log.Fatalf("Judge error: %v", err)
	}
}

func setupLogging(logFile string) error {
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		log.SetOutput(f)
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	}
	return nil
}

func startAPIServer(host string, port int, judge *core.Judge) error {
	// TODO: Implement HTTP API server for local testing
	// Endpoints:
	// - GET /status - Judge status
	// - POST /submit - Submit solution
	// - GET /submission/{id} - Get submission result
	return fmt.Errorf("not implemented")
}

func startProblemWatcher(judge *core.Judge) error {
	// TODO: Implement file system watcher for problem directories
	// When problems change, update the problem list sent to server
	return fmt.Errorf("not implemented")
}

func runSelfTests(judge *core.Judge) error {
	// TODO: Run self-tests for all available executors
	// Compile and run "Hello World" for each language
	return fmt.Errorf("not implemented")
}
