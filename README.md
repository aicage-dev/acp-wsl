# acp-wsl

`acp-wsl` lets a Windows IDE plugin communicate over ACP with an agent running in WSL.

It is an ACP stdio proxy that translates ACP filesystem paths between Windows drive paths and the corresponding
`/mnt/<drive>` WSL paths. Install it on Windows or in WSL and route ACP traffic through it.

## Install

### Release binary

Download the release binary for your operating system and architecture from
[GitHub Releases](https://github.com/aicage/acp-wsl/releases).

1. Download from [latest release](https://github.com/aicage/acp-wsl/releases/latest).
2. Rename and place it on `PATH`.
   - on Windows, rename to `acp-wsl.exe`.
   - on WSL, rename to `acp-wsl` and make executable.

### Build from source

To build from source instead, use the Go version declared in `go.mod`:

```text
go build ./cmd/acp-wsl
```

## Run an ACP agent in WSL from a Windows IDE

```text
acp-wsl [--unknown-fields=false] -- <agent-command> [agent-arguments]
```

Run the proxy on either side of the boundary:

| Proxy location | Command                                                  |
|----------------|----------------------------------------------------------|
| Windows        | `acp-wsl -- wsl --distribution Ubuntu -- your-acp-agent` |
| WSL            | `acp-wsl -- your-acp-agent`                              |

The command after `--` is started by `acp-wsl`. ACP messages pass between the Windows client and that command over
standard input and output.

## Options

### --unknown-fields

`--unknown-fields` defaults to `true`. For ACP messages that do not match the embedded schema, it also maps an
absolute Windows or WSL path in unrecognized payload fields. ACP permits agents to send custom fields, which may
contain paths not covered by the schema.

Set `--unknown-fields=false` to disable mapping outside the embedded ACP schema:

```text
acp-wsl --unknown-fields=false -- <agent-command> [agent-arguments]
```

When enabled, unknown-field mapping recognizes only absolute `C:\...`, `C:/...`, and `/mnt/c/...` values. Documented
ACP path fields remain mapped. Free text, command arguments, environment values, and non-`file` URLs are not candidates.

### --version

`--version` prints the installed version and exits.

### --help

`--help` prints command usage and exits.

## Path mapping

| Windows client → WSL agent                               | WSL agent → Windows client                               |
|----------------------------------------------------------|----------------------------------------------------------|
| `C:\work\project` → `/mnt/c/work/project`                | `/mnt/c/work/project` → `C:\work\project`                |
| `file:///C:/work/project` → `file:///mnt/c/work/project` | `file:///mnt/c/work/project` → `file:///C:/work/project` |

Mapped fields in official ACP are defined in:

- [v1](internal/transform/pathrules/v1.json): see [ACP v1 filesystem field inventory](doc/protocol/v1-path-fields.md)
- [v2](internal/transform/pathrules/v2.json): see [ACP v2 filesystem field inventory](doc/protocol/v2-path-fields.md)

ACP v1 and v2 filesystem fields and typed `file:` URIs are mapped.

UNC paths, drive-relative paths such as `C:work`, and current-drive-rooted paths such as `\work` are forwarded
unchanged; the proxy cannot determine their absolute location.

## IDE setup examples

Here are configuration examples for the communication of IDE plugins on Windows to agents in WSL.

1. Install the IDE plugin.
2. Install `acp-wsl` either on Windows or in WSL.
3. Copy the entry for the agent and proxy location you want into the plugin configuration.

<!-- markdownlint-disable line-length -->
| IDE       | Plugin                                                                                     | Example configuration                                                |
|-----------|--------------------------------------------------------------------------------------------|----------------------------------------------------------------------|
| VS Code   | [ACP Client](https://marketplace.visualstudio.com/items?itemName=formulahendry.acp-client) | [settings.json](doc/example/config/ide-plugins/VScode/settings.json) |
| JetBrains | [AI Assistant](https://www.jetbrains.com/help/ai-assistant/activate-agents.html)           | [acp.json](doc/example/config/ide-plugins/JetBrains/acp.json)        |
<!-- markdownlint-enable line-length -->

> The agent and its adapters should run with WSL's Node.js and npm — not node/npm in Windows inherited through `PATH`.

### ACP adapters

Also install the ACP adapter for these agents:

- Amp: [amp-acl](https://github.com/tao12345666333/amp-acp)
- Claude: [claude-agent-acp](https://github.com/agentclientprotocol/claude-agent-acp)
- Codex: [codex-acp](https://github.com/agentclientprotocol/codex-acp/)

## Debugging

`acp-wsl-debug` behaves like `acp-wsl` with the additional option `--log-file`.

```text
acp-wsl-debug --log-file <path> -- <agent-command> [agent-arguments]
```

`--log-file <path>` logs forwarded ACP traffic before and after mapping, mapping decisions, stream errors, and agent
process exit status.

> Warning: The log contains raw ACP traffic, including API keys, credentials, prompts, source paths, and agent output.

Download the debug command from the latest release
([acp-wsl-debug.zip](https://github.com/aicage/acp-wsl/releases/latest/download/acp-wsl-debug.zip)) or build it from
source:

```text
go build ./cmd/acp-wsl-debug
```
