package transform

import "testing"

func TestJSONRPCLifecycleEdgeCases(t *testing.T) {
	t.Run("nonstandard request ID does not correlate response", func(t *testing.T) {
		mapper := New(false)
		request := []byte(`{"jsonrpc":"2.0","id":true,"method":"session/list","params":{"cwd":"C:\\work"}}`)
		got, _ := mapper.Transform(request, ToLinux)
		wantRequest := []byte(`{"jsonrpc":"2.0","id":true,"method":"session/list","params":{"cwd":"/mnt/c/work"}}`)
		if !sameJSON(got, wantRequest) {
			t.Fatalf("request = %s, want %s", got, wantRequest)
		}

		response := []byte(`{"jsonrpc":"2.0","id":true,"result":{"sessions":[{"cwd":"/mnt/c/work"}]}}`)
		got, _ = mapper.Transform(response, ToWindows)
		if !sameJSON(got, response) {
			t.Fatalf("response = %s, want unchanged %s", got, response)
		}
	})

	t.Run("uncorrelated response is unchanged", func(t *testing.T) {
		response := []byte(`{"jsonrpc":"2.0","id":"missing","result":{"sessions":[{"cwd":"/mnt/c/work"}]}}`)
		got, _ := New(false).Transform(response, ToWindows)
		if !sameJSON(got, response) {
			t.Fatalf("response = %s, want unchanged %s", got, response)
		}
	})

	t.Run("request without params can still correlate its response", func(t *testing.T) {
		mapper := New(false)
		_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"session/list"}`), ToLinux)
		response := []byte(`{"jsonrpc":"2.0","id":1,"result":{"sessions":[{"cwd":"/mnt/c/work"}]}}`)
		got, _ := mapper.Transform(response, ToWindows)
		want := []byte(`{"jsonrpc":"2.0","id":1,"result":{"sessions":[{"cwd":"C:\\work"}]}}`)
		if !sameJSON(got, want) {
			t.Fatalf("response = %s, want %s", got, want)
		}
	})

	for _, initialize := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":3}}`,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2"}}`,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2.5}}`,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":null}`,
	} {
		t.Run("invalid version retains v1 routing", func(t *testing.T) {
			mapper := New(false)
			_, _ = mapper.Transform([]byte(initialize), ToLinux)
			fork := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/fork","params":{"cwd":"C:\\work"}}`)
			got, _ := mapper.Transform(fork, ToLinux)
			if !sameJSON(got, fork) {
				t.Fatalf("fork = %s, want unchanged %s", got, fork)
			}
		})
	}

	t.Run("initialize response negotiates version", func(t *testing.T) {
		mapper := New(false)
		_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1}}`), ToLinux)
		_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":2}}`), ToWindows)
		fork := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/fork","params":{"cwd":"C:\\work"}}`)
		got, _ := mapper.Transform(fork, ToLinux)
		want := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/fork","params":{"cwd":"/mnt/c/work"}}`)
		if !sameJSON(got, want) {
			t.Fatalf("fork = %s, want %s", got, want)
		}
	})

	t.Run("non-command v2 permission subject is unchanged", func(t *testing.T) {
		mapper := New(false)
		_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
		permission := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/request_permission","params":{"subject":{"type":"file","cwd":"C:\\work"}}}`)
		got, _ := mapper.Transform(permission, ToLinux)
		if !sameJSON(got, permission) {
			t.Fatalf("permission = %s, want unchanged %s", got, permission)
		}
	})
}

func TestSessionResponsePathsUseRequestCorrelation(t *testing.T) {
	tests := []struct {
		name, request, response, want string
		version                       int
	}{
		{
			name:     "list response",
			request:  `{"jsonrpc":"2.0","id":"list","method":"session/list","params":{}}`,
			response: `{"jsonrpc":"2.0","id":"list","result":{"sessions":[{"cwd":"/mnt/c/work","additionalDirectories":["/mnt/d/lib"]}]}}`,
			want:     `{"jsonrpc":"2.0","id":"list","result":{"sessions":[{"cwd":"C:\\work","additionalDirectories":["D:\\lib"]}]}}`,
			version:  1,
		},
		{
			name:     "new session skill path",
			request:  `{"jsonrpc":"2.0","id":"new","method":"session/new","params":{"cwd":"C:\\work"}}`,
			response: `{"jsonrpc":"2.0","id":"new","result":{"sessionId":"session","availableCommands":[{"type":"skill","name":"review","description":"Review files","path":"/mnt/c/skills/review/SKILL.md"}]}}`,
			want:     `{"jsonrpc":"2.0","id":"new","result":{"sessionId":"session","availableCommands":[{"type":"skill","name":"review","description":"Review files","path":"C:\\skills\\review\\SKILL.md"}]}}`,
			version:  2,
		},
		{
			name:     "resume session skill path",
			request:  `{"jsonrpc":"2.0","id":"resume","method":"session/resume","params":{"sessionId":"session","cwd":"C:\\work"}}`,
			response: `{"jsonrpc":"2.0","id":"resume","result":{"availableCommands":[{"type":"skill","name":"review","description":"Review files","path":"/mnt/c/skills/review/SKILL.md"}]}}`,
			want:     `{"jsonrpc":"2.0","id":"resume","result":{"availableCommands":[{"type":"skill","name":"review","description":"Review files","path":"C:\\skills\\review\\SKILL.md"}]}}`,
			version:  2,
		},
		{
			name:     "fork session skill path",
			request:  `{"jsonrpc":"2.0","id":"fork","method":"session/fork","params":{"sessionId":"source","cwd":"C:\\work"}}`,
			response: `{"jsonrpc":"2.0","id":"fork","result":{"sessionId":"child","availableCommands":[{"type":"skill","name":"review","description":"Review files","path":"/mnt/c/skills/review/SKILL.md"}]}}`,
			want:     `{"jsonrpc":"2.0","id":"fork","result":{"sessionId":"child","availableCommands":[{"type":"skill","name":"review","description":"Review files","path":"C:\\skills\\review\\SKILL.md"}]}}`,
			version:  2,
		},
		{
			name:     "options skill path",
			request:  `{"jsonrpc":"2.0","id":"options","method":"session/options","params":{"cwd":"C:\\work"}}`,
			response: `{"jsonrpc":"2.0","id":"options","result":{"availableCommands":[{"type":"skill","name":"review","description":"Review files","path":"/mnt/c/skills/review/SKILL.md"}]}}`,
			want:     `{"jsonrpc":"2.0","id":"options","result":{"availableCommands":[{"type":"skill","name":"review","description":"Review files","path":"C:\\skills\\review\\SKILL.md"}]}}`,
			version:  2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapper := New(true)
			if test.version == 2 {
				mapper.Transform([]byte(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
			}
			validateACPFixture(t, test.version, []byte(test.request))
			mapper.Transform([]byte(test.request), ToLinux)
			validateACPFixture(t, test.version, []byte(test.response))
			got, _ := mapper.Transform([]byte(test.response), ToWindows)
			if !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s", got, test.want)
			}
		})
	}
}
