package transform

import "testing"

func TestContentFileURIsAndV2Diffs(t *testing.T) {
	mapper := New(true)
	_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
	input := []byte(`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"tool_call_update","locations":[{"path":"/mnt/c/a.go"}],"content":[{"type":"diff","changes":[{"operation":"move","oldPath":"/mnt/c/old.go","path":"/mnt/d/new.go"}]},{"type":"content","content":{"type":"resource_link","name":"x","uri":"file:///mnt/c/readme.md"}}]}}}`)
	want := `{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"tool_call_update","locations":[{"path":"C:\\a.go"}],"content":[{"type":"diff","changes":[{"operation":"move","oldPath":"C:\\old.go","path":"D:\\new.go"}]},{"type":"content","content":{"type":"resource_link","name":"x","uri":"file:///C:/readme.md"}}]}}}`
	got, _ := mapper.Transform(input, ToWindows)
	if !sameJSON(got, []byte(want)) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV2CommandPermissionCWD(t *testing.T) {
	mapper := New(true)
	mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
	input := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/request_permission","params":{"sessionId":"session","title":"Run command","options":[{"optionId":"allow","name":"Allow once","kind":"allow_once"}],"subject":{"type":"command","command":"pwd","cwd":"C:\\work"}}}`)
	validateACPFixture(t, 2, input)
	want := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/request_permission","params":{"sessionId":"session","title":"Run command","options":[{"optionId":"allow","name":"Allow once","kind":"allow_once"}],"subject":{"type":"command","command":"pwd","cwd":"/mnt/c/work"}}}`)
	got, _ := mapper.Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV2ForkMCPStdioCommandPath(t *testing.T) {
	mapper := New(true)
	mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
	input := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/fork","params":{"sessionId":"source","cwd":"C:\\work","mcpServers":[{"type":"stdio","name":"local","command":"C:\\tools\\server.exe","args":[]}]}}`)
	validateACPFixture(t, 2, input)
	want := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/fork","params":{"sessionId":"source","cwd":"/mnt/c/work","mcpServers":[{"type":"stdio","name":"local","command":"/mnt/c/tools/server.exe","args":[]}]}}`)
	got, _ := mapper.Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV2PlanFileURI(t *testing.T) {
	mapper := New(true)
	mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
	input := []byte(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"session","update":{"sessionUpdate":"plan_update","plan":{"type":"file","planId":"plan","uri":"file:///C:/work/plan.md"}}}}`)
	validateACPFixture(t, 2, input)
	want := []byte(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"session","update":{"sessionUpdate":"plan_update","plan":{"type":"file","planId":"plan","uri":"file:///mnt/c/work/plan.md"}}}}`)
	got, _ := mapper.Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestSessionUpdateContentAndPassthroughVariants(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{
			"message chunk content",
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"agent_message_chunk","content":{"type":"image","uri":"file:///C:/work/chunk.png"}}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"agent_message_chunk","content":{"type":"image","uri":"file:///mnt/c/work/chunk.png"}}}}`,
		},
		{
			"message list content",
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"agent_message","content":[{"type":"resource","resource":{"uri":"file:///C:/work/result.txt"}}]}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"agent_message","content":[{"type":"resource","resource":{"uri":"file:///mnt/c/work/result.txt"}}]}}}`,
		},
		{
			"future update is unchanged",
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"future_update","path":"C:\\work"}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"future_update","path":"C:\\work"}}}`,
		},
		{
			"non-file plan is unchanged",
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"plan_update","plan":{"type":"text","uri":"file:///C:/work/plan.md"}}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"plan_update","plan":{"type":"text","uri":"file:///C:/work/plan.md"}}}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapper := New(false)
			_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
			got, _ := mapper.Transform([]byte(test.input), ToLinux)
			if !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s", got, test.want)
			}
		})
	}
}
