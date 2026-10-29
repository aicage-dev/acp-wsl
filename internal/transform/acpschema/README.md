# Embedded ACP schemas

[`.github/versions/acp-schema.yml`](../../../.github/versions/acp-schema.yml) records the ACP schema releases
embedded by this version of `acp-wsl`.

`v1.json` is the `schema.json` asset for v1; `v2.json` is the `schema.unstable.json` asset for v2.

CI downloads those release assets and checks that they exactly match the embedded copies.

Releases include the manifest as `acp-schema-versions.yml`.
