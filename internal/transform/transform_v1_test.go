package transform

import "testing"

func TestPathsInKnownV1MethodFields(t *testing.T) {
	tests := []struct {
		name, input, want string
		direction         Direction
	}{
		{"new session", `{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work\\project","additionalDirectories":["D:\\lib","src\\local.go"]}}`, `{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/mnt/c/work/project","additionalDirectories":["/mnt/d/lib","src/local.go"]}}`, ToLinux},
		{"new session trailing separator", `{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work\\"}}`, `{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/mnt/c/work/"}}`, ToLinux},
		{"read text", `{"jsonrpc":"2.0","id":1,"method":"fs/read_text_file","params":{"path":"/mnt/c/work/file.go"}}`, `{"jsonrpc":"2.0","id":1,"method":"fs/read_text_file","params":{"path":"C:\\work\\file.go"}}`, ToWindows},
		{"terminal", `{"jsonrpc":"2.0","id":1,"method":"terminal/create","params":{"cwd":"/mnt/d/work"}}`, `{"jsonrpc":"2.0","id":1,"method":"terminal/create","params":{"cwd":"D:\\work"}}`, ToWindows},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := New(true).Transform([]byte(test.input), test.direction)
			if !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s", got, test.want)
			}
		})
	}
}

func TestV1SessionUpdateKnownPathMappings(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{
			"markdown plan path",
			`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"session","update":{"sessionUpdate":"plan_update","plan":{"type":"markdown","planId":"plan","content":"# Plan","path":"C:\\work\\plan.md"}}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"session","update":{"sessionUpdate":"plan_update","plan":{"type":"markdown","planId":"plan","content":"# Plan","path":"/mnt/c/work/plan.md"}}}}`,
		},
		{
			"subagent workspace paths",
			`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"parent","update":{"sessionUpdate":"subagent_update","sessionId":"child","cwd":"C:\\work\\child","additionalDirectories":["D:\\shared"]}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"parent","update":{"sessionUpdate":"subagent_update","sessionId":"child","cwd":"/mnt/c/work/child","additionalDirectories":["/mnt/d/shared"]}}}`,
		},
		{
			"session message chunk content",
			`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"parent","update":{"sessionUpdate":"session_message_chunk","messageId":"message","content":{"type":"image","data":"aW1hZ2U=","mimeType":"image/png","uri":"file:///C:/work/chunk.png"}}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"parent","update":{"sessionUpdate":"session_message_chunk","messageId":"message","content":{"type":"image","data":"aW1hZ2U=","mimeType":"image/png","uri":"file:///mnt/c/work/chunk.png"}}}}`,
		},
		{
			"session message content",
			`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"parent","update":{"sessionUpdate":"session_message","messageId":"message","content":[{"type":"resource","resource":{"uri":"file:///C:/work/result.txt","text":"result"}}]}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"parent","update":{"sessionUpdate":"session_message","messageId":"message","content":[{"type":"resource","resource":{"uri":"file:///mnt/c/work/result.txt","text":"result"}}]}}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validateACPFixture(t, 1, []byte(test.input))
			got, _ := New(true).Transform([]byte(test.input), ToLinux)
			if !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s", got, test.want)
			}
		})
	}
}

func TestV1PermissionToolCallPaths(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","id":4,"method":"session/request_permission","params":{"toolCall":{"locations":[{"path":"/mnt/c/file.go"}],"content":[{"type":"diff","path":"/mnt/c/file.go","newText":"x"}]}}}`)
	got, _ := New(true).Transform(input, ToWindows)
	want := []byte(`{"jsonrpc":"2.0","id":4,"method":"session/request_permission","params":{"toolCall":{"locations":[{"path":"C:\\file.go"}],"content":[{"type":"diff","path":"C:\\file.go","newText":"x"}]}}}`)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV1MCPStdioCommandPath(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work","mcpServers":[{"name":"local","command":"C:\\tools\\server.exe","args":[],"env":[]}]}}`)
	validateACPFixture(t, 1, input)
	want := []byte(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/mnt/c/work","mcpServers":[{"name":"local","command":"/mnt/c/tools/server.exe","args":[],"env":[]}]}}`)
	got, _ := New(true).Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV1MCPHTTPCommandExtensionPassesThrough(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work","mcpServers":[{"type":"http","name":"remote","url":"https://example.test/mcp","headers":[],"command":"C:\\must-not-change.exe"}]}}`)
	validateACPFixture(t, 1, input)
	want := []byte(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/mnt/c/work","mcpServers":[{"type":"http","name":"remote","url":"https://example.test/mcp","headers":[],"command":"C:\\must-not-change.exe"}]}}`)
	got, _ := New(false).Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}
