# SLM Integration — Future Improvements

## Motivation

Replace LLM calls with custom-built Small Language Models (SLMs) that run
locally on CPU. Each SLM is a tiny transformer (~500K–5M params) trained from
scratch for a single well-scoped task. The SLM runs as a standalone Go HTTP
service that any project can call via REST API.

Note: Still thinking about this, may not do it.

### Architecture

| Component | Spec |
|---|---|
| Tokenization | Byte-level (no tokenizer dependency) |
| Layers | 2–4 transformer layers |
| Heads | 4–8 |
| Embedding dim | 64–128 |
| FF dim | 256–512 |
| Total params | ~500K–5M |
| Model size | ~2–20 MB (fp32) |
| Inference | <10ms on CPU |
| Service | Standalone Go HTTP API |
| Client | Any project calls via REST (`POST /api/v1/predict`) |

## Use Case: Secret Scan

- **Where**: `orchestrator/internal/validation/diffreview.go` (validation gate)
- **Now**: Regex-only — catches known patterns (API keys, tokens, private keys)
- **SLM**: Classify diff lines as containing secrets with higher accuracy and
  fewer false positives than regex patterns. Catches obfuscated or
  unconventional secret formats.

## Inference Engine Integration

The SLM runs as a standalone Go HTTP service, loaded with model files from a
configurable path at startup. Any project calls it via REST:

```go
// Client in any language
POST /api/v1/predict
Content-Type: application/json

{"model": "secret-scan", "input": "..."}

---

200 OK
Content-Type: application/json

{"output": "secret_detected"}
```

The orchestrator calls the SLM service directly over HTTP as a validation gate.
kai-code (C# .NET) calls it as an HTTP client — no embedded inference engine
needed.

## Training Data Pipeline

The observability layer already logs every prompt, result, diff, and build
output. Training data for each SLM is generated from:

1. **Historical LLM runs** — extract input/output pairs from audit logs.
2. **Self-supervised generation** — run the existing system on open-source
   repos, collect inputs, use LLM outputs as ground truth.
3. **Synthetic augmentation** — create additional training examples from
   code corpora.
