# External Gate Plugin System

Custom validation gates can be built as standalone binaries in any language. The host discovers them via manifest files and executes them via subprocess.

---

## How It Works

```
~/.kai/plugins/
  my-gate/
    plugin.json         # manifest
    my-gate-binary      # executable (Go, Rust, Python, shell, etc.)
```

The orchestrator scans `~/.kai/plugins/` (or `$KAI_PLUGIN_DIR`) on startup, reads each `plugin.json`, and registers the binary as a pipeline gate.

---

## Manifest (`plugin.json`)

```json
{
  "name": "my_gate",
  "version": "1.0.0",
  "api_version": "v1",
  "type": "gate",
  "gate_types": ["my_gate"],
  "binary": "my-gate-binary",
  "config": {
    "threshold": "high"
  }
}
```

| Field | Description |
|---|---|
| `name` | Unique identifier for the plugin |
| `type` | Must be `"gate"` |
| `gate_types` | Gate type names that this plugin handles (used in pipeline YAML under `gates:`) |
| `binary` | Relative path to the executable inside the plugin directory |
| `config` | Custom key-value pairs passed as CLI flags to the binary |

---

## Binary Contract

The host invokes the binary with CLI flags and expects a JSON result on stdout.

### Invocation

```
my-gate-binary \
  --repo-dir=/path/to/repo \
  --workspace-dir=/path/to/workspace \
  --branch=feat/my-change \
  --exit-code=0 \
  --threshold=high
```

All `config` fields from the manifest are passed as `--key=value` flags.

### Expected Output (stdout)

```json
{"status": "passed", "message": "everything looks good"}
```

| Status | Meaning |
|---|---|
| `passed` | Gate passed |
| `failed` | Gate failed — pipeline stops |
| `skipped` | Not applicable (e.g. no tool found) |
| `pending` | Waiting on external input |

### Exit Codes

- **Exit 0** — result is read from stdout JSON
- **Non-zero** — gate is marked `failed` with stderr captured as the message

---

## Example (Go)

The project at `kai-plugins/go/conventional-commits-gate/` is a working reference:

```
kai-plugins/go/conventional-commits-gate/
├── go.mod
├── main.go
└── plugin.json
```

It checks that recent commit messages follow conventional commits format.

**Build and install:**

```bash
cd kai-plugins
make install
```

This compiles the binary and copies it to `~/.kai/plugins/conventional-commits/`.

**Use in a pipeline YAML:**

```yaml
steps:
  - id: "build"
    gates:
      - exit_zero
      - lint
      - conventional_commits
```

---

## Pipeline YAML Reference

Any gate type name listed in `gate_types` in the manifest can be used in any step:

```yaml
steps:
  - id: "step-1"
    prompt: "Implement the feature"
    gates:
      - exit_zero
      - my_custom_gate      # matches gate_types in plugin.json
```

Multiple gates run sequentially. All must pass for the step to succeed.
