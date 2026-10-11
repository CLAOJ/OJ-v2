package moss

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type received struct {
	header []string
	files  map[string]string
}

// fakeServer speaks the MOSS server side once and records what it got.
func fakeServer(t *testing.T, acceptLanguage bool) (string, <-chan received) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	out := make(chan received, 1)
	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		got := received{files: map[string]string{}}
		for i := 0; i < 6; i++ { // moss, directory, X, maxmatches, show, language
			line, _ := r.ReadString('\n')
			got.header = append(got.header, strings.TrimSpace(line))
		}
		if !acceptLanguage {
			fmt.Fprint(conn, "no\n")
			out <- got
			return
		}
		fmt.Fprint(conn, "yes\n")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "file ") {
				var id, size int
				var lang, name string
				fmt.Sscanf(line, "file %d %s %d %s", &id, &lang, &size, &name)
				buf := make([]byte, size)
				_, _ = io.ReadFull(r, buf)
				got.files[name] = string(buf)
				continue
			}
			if strings.HasPrefix(line, "query 0 ") {
				fmt.Fprint(conn, "http://moss.stanford.edu/results/1/2345\n")
				continue
			}
			if line == "end" {
				break
			}
		}
		out <- got
	}()
	return ln.Addr().String(), out
}

func TestSubmit_SpeaksTheProtocol(t *testing.T) {
	addr, got := fakeServer(t, true)
	c := &Client{UserID: "987654", Addr: addr, Show: 200}
	url, err := c.Submit("cc", "tik25 - knap", []File{
		{Name: "alice", Content: []byte("int main(){}\n")},
		{Name: "bob smith", Content: []byte("x")},
	})
	require.NoError(t, err)
	require.Equal(t, "http://moss.stanford.edu/results/1/2345", url)

	r := <-got
	require.Equal(t, []string{"moss 987654", "directory 0", "X 0", "maxmatches 10", "show 200", "language cc"}, r.header)
	require.Equal(t, "int main(){}\n", r.files["alice"])
	require.Equal(t, "x", r.files["bob_smith"], "spaces in names become underscores")
}

func TestSubmit_LanguageRejected(t *testing.T) {
	addr, _ := fakeServer(t, false)
	_, err := (&Client{UserID: "1", Addr: addr}).Submit("klingon", "c", []File{{Name: "a", Content: []byte("x")}})
	require.ErrorIs(t, err, ErrLanguageRejected)
}

func TestSubmit_ConnectionRefused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close()
	_, err = (&Client{UserID: "1", Addr: addr}).Submit("cc", "c", []File{{Name: "a", Content: []byte("x")}})
	require.Error(t, err)
}
