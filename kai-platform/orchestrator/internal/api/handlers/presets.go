package handlers

var presets = []map[string]any{
	{
		"name":        "default",
		"description": "Balanced default policy with moderate timeouts and standard tool access",
		"policy": map[string]any{
			"allowed_dirs":       []string{"internal/", "src/"},
			"max_retries":        2,
			"retry_delay_seconds": 5,
			"retry_backoff":       "linear",
			"timeout_seconds":     600,
		},
		"yaml": `version: 1
project: my-project
output:
  type: pr
  branch_prefix: feat/

steps:
  - id: scaffold
    prompt: "Initialize project structure"
    policy:
      allowed_dirs:
        - internal/
        - src/
      max_retries: 2
      retry_delay_seconds: 5
      retry_backoff: linear
      timeout_seconds: 600
    validation:
      - exit_zero
  - id: implement
    prompt: "Implement the feature"
    depends_on:
      - scaffold
    policy:
      allowed_dirs:
        - internal/
        - src/
      max_retries: 2
      retry_delay_seconds: 5
      retry_backoff: linear
      timeout_seconds: 600
    validation:
      - exit_zero
      - lint
      - tests
`,
	},
	{
		"name":        "strict",
		"description": "Tight security with short timeouts, limited dirs, and tools",
		"policy": map[string]any{
			"allowed_dirs":       []string{"internal/"},
			"allowed_tools":      []string{"write", "read", "run"},
			"max_retries":        1,
			"retry_delay_seconds": 10,
			"timeout_seconds":     300,
		},
		"yaml": `version: 1
project: my-project
output:
  type: pr
  branch_prefix: feat/

steps:
  - id: scaffold
    prompt: "Initialize project structure"
    policy:
      allowed_dirs:
        - internal/
      allowed_tools:
        - write
        - read
        - run
      max_retries: 1
      retry_delay_seconds: 10
      timeout_seconds: 300
    validation:
      - exit_zero
  - id: implement
    prompt: "Implement the feature"
    depends_on:
      - scaffold
    policy:
      allowed_dirs:
        - internal/
      allowed_tools:
        - write
        - read
        - run
      max_retries: 1
      retry_delay_seconds: 10
      timeout_seconds: 300
    validation:
      - exit_zero
      - lint
      - tests
`,
	},
	{
		"name":        "relaxed",
		"description": "Permissive policy for rapid prototyping — long timeouts, many retries, no dir restrictions",
		"policy": map[string]any{
			"max_retries":        5,
			"retry_delay_seconds": 3,
			"retry_backoff":       "exponential",
			"timeout_seconds":     1800,
		},
		"yaml": `version: 1
project: my-project
output:
  type: branch
  branch_prefix: feat/

steps:
  - id: scaffold
    prompt: "Initialize project structure"
    policy:
      max_retries: 5
      retry_delay_seconds: 3
      retry_backoff: exponential
      timeout_seconds: 1800
    validation:
      - exit_zero
  - id: implement
    prompt: "Implement the feature"
    depends_on:
      - scaffold
    policy:
      max_retries: 5
      retry_delay_seconds: 3
      retry_backoff: exponential
      timeout_seconds: 1800
    validation:
      - exit_zero
      - lint
      - tests
`,
	},
}
