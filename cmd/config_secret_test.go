package cmd

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"
)

// fakeTerminal makes readSecret treat stdin as a terminal whose hidden input
// is secret, and records the descriptor it was read from.
func fakeTerminal(t *testing.T, secret string) *int {
	t.Helper()

	origIsTerminal, origReadPassword := isTerminal, readPassword
	t.Cleanup(func() { isTerminal, readPassword = origIsTerminal, origReadPassword })

	readFrom := new(int)
	*readFrom = -1
	isTerminal = func(int) bool { return true }
	readPassword = func(fd int) ([]byte, error) {
		*readFrom = fd
		return []byte(secret), nil
	}

	return readFrom
}

func TestReadSecret_ReadsWithoutEchoOnATerminal(t *testing.T) {
	readFrom := fakeTerminal(t, " s3cret ")

	stdin, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { _ = stdin.Close(); _ = w.Close() })

	var out bytes.Buffer
	got, err := readSecret(&out, stdin, bufio.NewReader(stdin))
	if err != nil {
		t.Fatalf("readSecret: %v", err)
	}

	if got != "s3cret" {
		t.Errorf("got %q, want %q", got, "s3cret")
	}
	if *readFrom != int(stdin.Fd()) {
		t.Errorf("read the secret from fd %d, want stdin's fd %d", *readFrom, stdin.Fd())
	}
	// The unechoed Enter is replaced with a newline, and nothing else is shown.
	if out.String() != "\n" {
		t.Errorf("output %q, want a single newline", out.String())
	}
}

func TestReadSecret_ReadsALineWhenNotATerminal(t *testing.T) {
	in := strings.NewReader("tok\nnext\n")
	reader := bufio.NewReader(in)

	got, err := readSecret(&bytes.Buffer{}, in, reader)
	if err != nil {
		t.Fatalf("readSecret: %v", err)
	}
	if got != "tok" {
		t.Errorf("got %q, want %q", got, "tok")
	}

	// The following prompt continues from the next line.
	if next, _ := reader.ReadString('\n'); next != "next\n" {
		t.Errorf("next line %q, want %q", next, "next\n")
	}
}
