package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesRawTrafficLog(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "traffic.log")
	t.Setenv("ACP_PATH_DEBUG_HELPER", "1")
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work"}}` + "\n")
	var output, stderr bytes.Buffer
	err := run([]string{"--log-file", logFile, "--", os.Args[0], "-test.run=TestHelperProcess", "--"}, input, &output, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(contents, []byte("client->agent")) || !bytes.Contains(contents, []byte("subprocess exited successfully")) {
		t.Fatalf("log missing required events: %s", contents)
	}
}

func TestRunRequiresLogFile(t *testing.T) {
	err := run([]string{"--", "agent"}, nil, io.Discard, io.Discard)
	if err == nil || err.Error() != "--log-file is required" {
		t.Fatalf("run error = %v, want --log-file is required", err)
	}
}

func TestHelperProcess(_ *testing.T) {
	if os.Getenv("ACP_PATH_DEBUG_HELPER") != "1" {
		return
	}
	line, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	if !bytes.Contains(line, []byte(`/mnt/c/work`)) {
		fmt.Fprintln(os.Stderr, "input was not transformed")
		os.Exit(3)
	}
	fmt.Fprintln(os.Stdout, `{"jsonrpc":"2.0","id":9,"method":"fs/read_text_file","params":{"path":"/mnt/c/work/result.go"}}`)
	os.Exit(0)
}
