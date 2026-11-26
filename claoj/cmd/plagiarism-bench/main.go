// Command plagiarism-bench runs the contest plagiarism + AI detection job
// against a local benchmark database (see scripts/plagiarism-bench).
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := flag.String("dsn", "", "MySQL DSN, e.g. root@tcp(127.0.0.1:3307)/claoj_bench?parseTime=true")
	contests := flag.String("contests", "", "comma-separated contest keys")
	out := flag.String("out", "", "report directory (reports land in <out>/<contest_id>)")
	py := flag.String("python", "", "compare50 venv python")
	flag.Parse()
	if *dsn == "" || *contests == "" || *out == "" || *py == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*dsn, strings.Split(*contests, ","), *out, *py); err != nil {
		fmt.Fprintln(os.Stderr, "plagiarism-bench:", err)
		os.Exit(1)
	}
}

func run(dsn string, keys []string, out, py string) error {
	jobs.PlagiarismDir = out
	jobs.Compare50Python = py
	cfg, err := gomysql.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("bad -dsn: %w", err)
	}
	if host, _, err := net.SplitHostPort(cfg.Addr); err != nil || (host != "127.0.0.1" && host != "localhost") {
		return fmt.Errorf("refusing non-local DSN host %q: the bench runs only against the local bench DB", cfg.Addr)
	}
	cfg.ParseTime = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{})
	if err != nil {
		return err
	}
	gen := jobs.LLMGeneratorFromEnv()
	for _, key := range keys {
		var c models.Contest
		if err := db.Where("`key` = ?", strings.TrimSpace(key)).First(&c).Error; err != nil {
			return fmt.Errorf("contest %q: %w", key, err)
		}
		n, err := jobs.RunContestMoss(context.Background(), db, c.ID, jobs.Compare50{}, gen)
		if err != nil {
			return fmt.Errorf("contest %q: %w", key, err)
		}
		fmt.Printf("%s (id %d): %d checks\n", key, c.ID, n)
	}
	return nil
}
