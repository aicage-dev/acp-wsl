package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestSessionLifecycleAndListPaths(t *testing.T) {
	tests := []struct {
		name, input, want string
		version           int
	}{
		{
			"v1 load",
			`{"jsonrpc":"2.0","id":1,"method":"session/load","params":{"sessionId":"session","cwd":"C:\\work","additionalDirectories":["D:\\lib"],"mcpServers":[]}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/load","params":{"sessionId":"session","cwd":"/mnt/c/work","additionalDirectories":["/mnt/d/lib"],"mcpServers":[]}}`,
			1,
		},
		{
			"v1 resume",
			`{"jsonrpc":"2.0","id":1,"method":"session/resume","params":{"sessionId":"session","cwd":"C:\\work","additionalDirectories":["D:\\lib"]}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/resume","params":{"sessionId":"session","cwd":"/mnt/c/work","additionalDirectories":["/mnt/d/lib"]}}`,
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
			`{"jsonrpc":"2.0","id":1,"method":"session/resume","params":{"sessionId":"session","cwd":"C:\\work","additionalDirectories":["D:\\lib"]}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/resume","params":{"sessionId":"session","cwd":"/mnt/c/work","additionalDirectories":["/mnt/d/lib"]}}`,
			2,
		},
		{
			"v2 list request",
			`{"jsonrpc":"2.0","id":1,"method":"session/list","params":{"cwd":"C:\\work"}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/list","params":{"cwd":"/mnt/c/work"}}`,
			2,
		},
		{
			"v2 options request",
			`{"jsonrpc":"2.0","id":1,"method":"session/options","params":{"cwd":"C:\\work","additionalDirectories":["D:\\lib"]}}`,
			`{"jsonrpc":"2.0","id":1,"method":"session/options","params":{"cwd":"/mnt/c/work","additionalDirectories":["/mnt/d/lib"]}}`,
			2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapper := New(true)
			if test.version == 2 {
				_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
			}
			validateACPFixture(t, test.version, []byte(test.input))
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

func sameJSON(first, second []byte) bool {
	var one, two any
	return json.Unmarshal(first, &one) == nil && json.Unmarshal(second, &two) == nil && reflect.DeepEqual(one, two)
}
