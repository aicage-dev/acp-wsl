# ACP v2 filesystem field inventory

This inventory is derived from the ACP v2 schema in the`agentclientprotocol/agent-client-protocol` repository.

It defines the schema-specific mappings.

The v2 router is separate from v1 because v2 changed tool-call diffs and terminal reporting.

<!-- markdownlint-disable line-length -->
| Context                                                   | Field                                                                                      |
|-----------------------------------------------------------|--------------------------------------------------------------------------------------------|
| `session/new`, `session/resume`, `session/fork`           | `params.cwd`, `params.additionalDirectories[]`                                             |
| `session/new`, `session/resume`, `session/fork` responses | `result.availableCommands[type=skill].path`                                                |
| `session/list` request                                    | `params.cwd`                                                                               |
| `session/list` response                                   | `result.sessions[].cwd`, `result.sessions[].additionalDirectories[]`                       |
| `session/options` request                                 | `params.cwd`, `params.additionalDirectories[]`                                             |
| `session/options` response                                | `result.availableCommands[type=skill].path`                                                |
| `session/prompt`                                          | typed resource/image file URIs                                                             |
| session lifecycle requests above                          | `params.mcpServers[]` stdio-server `command`                                               |
| `session/request_permission`                              | `params.subject.cwd` for a `command` subject                                               |
| `session/update` terminal update                          | `update.cwd`                                                                               |
| `session/update` subagent update                          | `update.cwd`, `update.additionalDirectories[]`                                             |
| `session/update` available-commands update                | `update.availableCommands[type=skill].path`                                                |
| `session/update` tool-call update                         | `locations[].path`; diff `changes[].path` and `changes[].oldPath`; typed content file URIs |
| `session/update` plan update                              | `update.plan.uri` for `file`; `update.plan.path` for `markdown`                            |
<!-- markdownlint-enable line-length -->

`file:` URIs in typed resource, image, and resource-link icon fields are converted.

HTTP URLs, elicitation URLs, and MCP URLs are forwarded unchanged. Other unmapped payload values, including extensions
and unknown tagged-union variants, are inspected by the default
[unknown fields](path-field-inventory.md#unknown-fields), which can be disabled.
