# Path-field inventory method

The v1 and v2 inventories are derived from the pinned ACP schema release assets downloaded by the `Test` workflow, not
from field-name searches alone.

For each schema version, inspect method parameters and results through their `$ref` chains. Include a field only when
the schema defines it as one of:

- a filesystem path, working directory, or workspace directory;
- a file URI in a typed protocol value; or
- an executable path, such as a stdio MCP server command.

The schema-specific pass maps the documented field within its known method and tagged-union variant.

## Unknown fields

By default, messages are validated against the embedded ACP v1 and v2 schemas. A second pass recursively inspects
only values in fields outside the selected schema branch. This supports clients and agents whose path fields are
outside ACP without maintaining per-agent mappings. Schema-defined fields are never candidates for this pass, even
when the bridge has no explicit mapping for them.

A whole string beginning with a Windows drive and slash (`C:\work` or `C:/work`) maps to `/mnt/c/work` toward Linux.
A whole `/mnt/<drive>` path maps back toward Windows. Relative paths, UNC paths, drive-relative paths, current-drive-rooted
paths, arbitrary Linux paths outside `/mnt/<drive>`, URLs, file URIs outside the typed schema mappings, and strings
containing NUL or line breaks are not converted. It does not search for paths embedded in prose.
JSON-RPC routing fields (`id`, `method`, and `jsonrpc`) and object keys remain unchanged.

This heuristic also applies when a whole-string path occurs in text, command arguments, or environment values outside
the schema. It does not check filesystem existence or container mounts. Existing schema mappings run after unknown
field conversion. Debug logs identify these conversions and their full field locations.

Both binaries enable this behavior through `--unknown-fields` by default. Set it to `false` before `--` to retain schema-only behavior:

```sh
acp-wsl --unknown-fields=false -- agent
acp-wsl-debug --log-file traffic.log --unknown-fields=false -- agent
```

When a schema release changes, update the embedded release asset, repeat the inventory review, update the versioned
inventory, and add a schema-validated mapping test for each newly included field.
