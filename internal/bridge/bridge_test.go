package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"testing"
)

func TestRunForwardsACPStreamsAndLogsTraffic(t *testing.T) {
	t.Setenv("ACP_PATH_BRIDGE_HELPER", "forward")
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work"}}` + "\n")
	var output, stderr, logOutput bytes.Buffer
	logger := log.New(&logOutput, "", 0)
	err := Run(input, &output, &stderr, logger, []string{os.Args[0], "-test.run=TestHelperProcess", "--"}, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"jsonrpc":"2.0","id":9,"method":"fs/read_text_file","params":{"path":"C:\\work\\result.go"}}` + "\n"
	if !sameJSON(output.Bytes(), []byte(want)) {
		t.Fatalf("stdout = %q, want %q", output.String(), want)
	}
	if !bytes.Contains(logOutput.Bytes(), []byte("client->agent raw=")) || !bytes.Contains(logOutput.Bytes(), []byte("agent->client raw=")) || !bytes.Contains(logOutput.Bytes(), []byte("subprocess exited successfully")) {
		t.Fatalf("log = %q, want both traffic directions and successful exit", logOutput.String())
	}
}

func TestRunReportsSubprocessFailure(t *testing.T) {
	t.Setenv("ACP_PATH_BRIDGE_HELPER", "fail")
	var output, stderr, logOutput bytes.Buffer
	logger := log.New(&logOutput, "", 0)
	err := Run(bytes.NewReader(nil), &output, &stderr, logger, []string{os.Args[0], "-test.run=TestHelperProcess", "--"}, true)
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	if !bytes.Contains(logOutput.Bytes(), []byte("subprocess exited with error:")) {
		t.Fatalf("log = %q, want subprocess failure", logOutput.String())
	}
}

func sameJSON(first, second []byte) bool {
	var one, two any
	return json.Unmarshal(first, &one) == nil && json.Unmarshal(second, &two) == nil && reflect.DeepEqual(one, two)
}

func TestHelperProcess(_ *testing.T) {
	switch os.Getenv("ACP_PATH_BRIDGE_HELPER") {
	case "forward":
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
	case "fail":
		os.Exit(7)
	}
}
