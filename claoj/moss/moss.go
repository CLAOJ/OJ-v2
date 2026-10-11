// Package moss is a client for the MOSS (Measure Of Software Similarity)
// server protocol, the same exchange as the official moss script, mosspy,
// and the DMOJ moss library v1 uses.
package moss

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const DefaultAddr = "moss.stanford.edu:7690"

// ErrLanguageRejected means the server does not support the language.
var ErrLanguageRejected = errors.New("moss: language not accepted by server")

// Client submits one batch of files per Submit call.
type Client struct {
	UserID     string        // MOSS account id (numeric string)
	Addr       string        // default DefaultAddr
	MaxMatches int           // -m: ignore code appearing in more files; default 10
	Show       int           // -n: matching file pairs to report; default 250
	Timeout    time.Duration // whole exchange; default 5 minutes
}

// File is one program; Name is how it appears in the report.
type File struct {
	Name    string
	Content []byte
}

// Submit uploads files in MOSS language lang (e.g. "cc", "java") and
// returns the report URL.
func (c *Client) Submit(lang, comment string, files []File) (string, error) {
	addr := c.Addr
	if addr == "" {
		addr = DefaultAddr
	}
	maxMatches, show, timeout := c.MaxMatches, c.Show, c.Timeout
	if maxMatches == 0 {
		maxMatches = 10
	}
	if show == 0 {
		show = 250
	}
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("moss: connect %s: %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)

	fmt.Fprintf(w, "moss %s\n", c.UserID)
	fmt.Fprintf(w, "directory 0\n")
	fmt.Fprintf(w, "X 0\n")
	fmt.Fprintf(w, "maxmatches %d\n", maxMatches)
	fmt.Fprintf(w, "show %d\n", show)
	fmt.Fprintf(w, "language %s\n", lang)
	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("moss: send header: %w", err)
	}
	reply, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("moss: read language reply: %w", err)
	}
	if strings.TrimSpace(reply) == "no" {
		fmt.Fprint(w, "end\n")
		_ = w.Flush()
		return "", fmt.Errorf("%w: %s", ErrLanguageRejected, lang)
	}

	for i, f := range files {
		name := strings.ReplaceAll(f.Name, " ", "_")
		fmt.Fprintf(w, "file %d %s %d %s\n", i+1, lang, len(f.Content), name)
		if _, err := w.Write(f.Content); err != nil {
			return "", fmt.Errorf("moss: upload %s: %w", name, err)
		}
	}
	fmt.Fprintf(w, "query 0 %s\n", comment)
	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("moss: send files: %w", err)
	}

	url, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("moss: read result: %w", err)
	}
	url = strings.TrimSpace(url)
	fmt.Fprint(w, "end\n")
	_ = w.Flush()
	if !strings.HasPrefix(url, "http") {
		return "", fmt.Errorf("moss: unexpected reply %q", url)
	}
	return url, nil
}
