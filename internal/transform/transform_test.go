package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

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

func TestSessionLifecycleAndListPaths(t *testing.T) {
	tests := []struct {
		name, input, want string
		version           int
	}{
		{
			"v1 load",
			`{"jsonrpc":"2.0","id":1,"method":"session/load","params":{"cwd":"C:\\work","additionalDirectories":["D:\\lib"]}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/load","params":{"cwd":"/mnt/c/work","additionalDirectories":["/mnt/d/lib"]}}`,
			1,
		},
		{
			"v1 resume",
			`{"jsonrpc":"2.0","id":1,"method":"session/resume","params":{"cwd":"C:\\work","additionalDirectories":["D:\\lib"]}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/resume","params":{"cwd":"/mnt/c/work","additionalDirectories":["/mnt/d/lib"]}}`,
			1,
		},
		{
			"v1 list request",
			`{"jsonrpc":"2.0","id":1,"method":"session/list","params":{"cwd":"C:\\work"}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/list","params":{"cwd":"/mnt/c/work"}}`,
			1,
		},
		{
			"v2 resume",
			`{"jsonrpc":"2.0","id":1,"method":"session/resume","params":{"cwd":"C:\\work","additionalDirectories":["D:\\lib"]}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/resume","params":{"cwd":"/mnt/c/work","additionalDirectories":["/mnt/d/lib"]}}`,
			2,
		},
		{
			"v2 list request",
			`{"jsonrpc":"2.0","id":1,"method":"session/list","params":{"cwd":"C:\\work"}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/list","params":{"cwd":"/mnt/c/work"}}`,
			2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapper := New(true)
			if test.version == 2 {
				_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
			}
			got, _ := mapper.Transform([]byte(test.input), ToLinux)
			if !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s", got, test.want)
			}
		})
	}
}

func TestPromptContentFileURIs(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"session/prompt","params":{"prompt":[{"type":"image","uri":"file:///C:/work/image.png"},{"type":"resource_link","uri":"file:///C:/work/readme.md","icons":[{"src":"file:///C:/work/icon.png"}]},{"type":"resource","resource":{"uri":"file:///C:/work/resource.txt"}}]}}`
	want := `{"jsonrpc":"2.0","id":1,"method":"session/prompt","params":{"prompt":[{"type":"image","uri":"file:///mnt/c/work/image.png"},{"type":"resource_link","uri":"file:///mnt/c/work/readme.md","icons":[{"src":"file:///mnt/c/work/icon.png"}]},{"type":"resource","resource":{"uri":"file:///mnt/c/work/resource.txt"}}]}}`
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			mapper := New(true)
			if version == 2 {
				_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
			}
			got, _ := mapper.Transform([]byte(input), ToLinux)
			if !sameJSON(got, []byte(want)) {
				t.Fatalf("got %s, want %s", got, want)
			}
		})
	}
}

func TestRemainingDocumentedPathMappings(t *testing.T) {
	tests := []struct {
		name, input, want string
		version           int
		direction         Direction
	}{
		{
			"v1 write text file",
			`{"jsonrpc":"2.0","id":1,"method":"fs/write_text_file","params":{"path":"/mnt/c/work/file.go"}}`,
			`{"jsonrpc":"2.0","id":1,"method":"fs/write_text_file","params":{"path":"C:\\work\\file.go"}}`,
			1,
			ToWindows,
		},
		{
			"v2 terminal update",
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"terminal_update","cwd":"C:\\work"}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"terminal_update","cwd":"/mnt/c/work"}}}`,
			2,
			ToLinux,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapper := New(true)
			if test.version == 2 {
				_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
			}
			got, _ := mapper.Transform([]byte(test.input), test.direction)
			if !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s", got, test.want)
			}
		})
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

func TestMalformedTypedContentPassesThrough(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{
			"prompt is not a content list",
			`{"jsonrpc":"2.0","id":1,"method":"session/prompt","params":{"prompt":"file:///C:/work/prompt.txt"}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/prompt","params":{"prompt":"file:///C:/work/prompt.txt"}}`,
		},
		{
			"malformed values beside a resource link",
			`{"jsonrpc":"2.0","id":1,"method":"session/prompt","params":{"prompt":[null,{"type":"text","text":"file:///C:/work/text.txt"},{"type":"resource","resource":"file:///C:/work/ignored.txt"},{"type":"resource_link","uri":"file:///C:/work/readme.md","icons":["file:///C:/work/icon.png"]}]}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/prompt","params":{"prompt":[null,{"type":"text","text":"file:///C:/work/text.txt"},{"type":"resource","resource":"file:///C:/work/ignored.txt"},{"type":"resource_link","uri":"file:///mnt/c/work/readme.md","icons":["file:///C:/work/icon.png"]}]}}`,
		},
		{
			"message chunk content is not an object",
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"agent_message_chunk","content":"file:///C:/work/chunk.txt"}}}`,
			`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"agent_message_chunk","content":"file:///C:/work/chunk.txt"}}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := New(false).Transform([]byte(test.input), ToLinux)
			if !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s", got, test.want)
			}
		})
	}
}

func TestToolCallRuleSkipsMalformedValues(t *testing.T) {
	input := []byte(`{"method":"session/update","params":{"update":{"sessionUpdate":"tool_call","toolCall":{"locations":[null,{"path":"C:\\work\\location.go"}],"content":{"type":"diff","path":"C:\\work\\current.go","changes":[null,{"oldPath":"C:\\work\\old.go","path":"C:\\work\\new.go"}]}}}}}`)
	want := []byte(`{"method":"session/update","params":{"update":{"sessionUpdate":"tool_call","toolCall":{"locations":[null,{"path":"/mnt/c/work/location.go"}],"content":{"type":"diff","path":"/mnt/c/work/current.go","changes":[null,{"oldPath":"/mnt/c/work/old.go","path":"/mnt/c/work/new.go"}]}}}}}`)
	got, _ := New(false).Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestPathAndURIConversionBoundaries(t *testing.T) {
	pathTests := []struct {
		name, input, want, kind string
		direction               Direction
	}{
		{"windows absolute", `C:\work`, "/mnt/c/work", "windows-drive", ToLinux},
		{"windows trailing separator", `C:\work\`, "/mnt/c/work/", "windows-drive", ToLinux},
		{"windows drive root", `C:\`, "/mnt/c/", "windows-drive", ToLinux},
		{"windows drive relative", `C:work`, `C:work`, "windows-drive-relative", ToLinux},
		{"windows UNC", `\\server\share`, `\\server\share`, "unc", ToLinux},
		{"windows current drive rooted", `\work`, `\work`, "windows-current-drive-rooted", ToLinux},
		{"windows relative", `src\work`, "src/work", "relative", ToLinux},
		{"linux absolute on linux side", "/home/work", "/home/work", "", ToLinux},
		{"WSL mount", "/mnt/c/work", `C:\work`, "wsl-mount", ToWindows},
		{"WSL mount trailing separator", "/mnt/c/work/", `C:\work\`, "wsl-mount", ToWindows},
		{"WSL mount root", "/mnt/c/", `C:\`, "wsl-mount", ToWindows},
		{"linux relative", "src/work", `src\work`, "relative", ToWindows},
		{"non-WSL linux absolute", "/home/work", "/home/work", "", ToWindows},
	}
	for _, test := range pathTests {
		t.Run(test.name, func(t *testing.T) {
			got, decision := convertPath(test.input, test.direction)
			if got != test.want || decision.Kind != test.kind {
				t.Fatalf("convertPath(%q) = (%q, %+v), want (%q, kind %q)", test.input, got, decision, test.want, test.kind)
			}
		})
	}

	uriTests := []struct {
		name, input, want, kind string
		direction               Direction
	}{
		{"windows file URI", "file:///C:/work/readme.md", "file:///mnt/c/work/readme.md", "file-uri", ToLinux},
		{"windows file URI trailing separator", "file:///C:/work/", "file:///mnt/c/work/", "file-uri", ToLinux},
		{"windows drive-root URI", "file:///C:/", "file:///mnt/c/", "file-uri", ToLinux},
		{"WSL file URI", "file:///mnt/c/work/readme.md", "file:///C:/work/readme.md", "file-uri", ToWindows},
		{"WSL file URI trailing separator", "file:///mnt/c/work/", "file:///C:/work/", "file-uri", ToWindows},
		{"WSL mount-root URI", "file:///mnt/c/", "file:///C:/", "file-uri", ToWindows},
		{"non-file URI", "https://example.test/work", "https://example.test/work", "", ToLinux},
		{"malformed URI", "%", "%", "", ToLinux},
		{"file URI with host", "file://server/share", "file://server/share", "", ToWindows},
		{"non-WSL file URI", "file:///home/work", "file:///home/work", "", ToWindows},
	}
	for _, test := range uriTests {
		t.Run(test.name, func(t *testing.T) {
			got, decision := convertURI(test.input, test.direction)
			if got != test.want || decision.Kind != test.kind {
				t.Fatalf("convertURI(%q) = (%q, %+v), want (%q, kind %q)", test.input, got, decision, test.want, test.kind)
			}
		})
	}
}

func TestUnsupportedMalformedAndAmbiguousValuesPassThrough(t *testing.T) {
	mapper := New(false)
	for _, input := range [][]byte{
		[]byte(`{"jsonrpc":"2.0","method":"_extension","params":{"path":"C:\\must-not-change"}}`),
		[]byte(`{"jsonrpc":"2.0","method":"session/new","params":{"cwd":"C:relative\\file"}}`),
		[]byte(`{"jsonrpc":"2.0","method":"session/new","params":{"cwd":"\\rooted"}}`),
		[]byte(`{"jsonrpc":"2.0","method":"session/new","params":{"cwd":"\\\\server\\share"}}`),
		[]byte(`not json`),
	} {
		got, _ := mapper.Transform(input, ToLinux)
		if !bytes.Equal(got, input) {
			t.Fatalf("got %q, want unchanged %q", got, input)
		}
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

func sameJSON(first, second []byte) bool {
	var one, two any
	return json.Unmarshal(first, &one) == nil && json.Unmarshal(second, &two) == nil && reflect.DeepEqual(one, two)
}
