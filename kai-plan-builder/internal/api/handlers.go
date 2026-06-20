package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	sdkgokit "kaiplatform.com/observability-sdk"
	"github.com/kaiplatform/plan-builder/internal/chat"
	"github.com/kaiplatform/plan-builder/internal/llm"
	"gopkg.in/yaml.v3"
)

const pipelineSchema = `Pipeline YAML schema:

version: 1                          # required, must be integer 1
project: <string>                   # required: project name

repo:                               # optional — omit for no VCS
  url: <string>                     # git remote URL
  base_branch: <string>             # default: main
  provider: <string>                # optional: forgejo, github, gitlab, bitbucket
  token_ref: <string>               # optional: secret reference for auth

output:                             # optional — defaults if omitted
  type: pr | branch | direct        # default: pr
  branch_prefix: <string>           # default: feat/

steps:                              # required — array, at least 1
  - id: <string>                    # required: unique step identifier
    prompt: |                       # required: instructions for agent
      <multiline text>
    depends_on:                     # optional — array of step IDs
      - <step-id>
    validation:                     # optional — array of gates
      - exit_zero                   # verify exit code 0
      - lint                        # run linter
      - typecheck                   # run type checker
      - tests                       # run test suite
      - diff_review                 # human review of diff
    approval: optional | required   # optional — default: optional
    policy:                         # optional — step constraints
      allowed_dirs:                 # optional — restrict file access
        - <path>
      allowed_tools:                # optional — restrict agent tools
        - read_file | write_file | run | glob | search | list_dir
      allowed_commands:             # optional — restrict shell commands
        - <command-glob>
      agent: <string>               # optional — pin to specific agent
      max_retries: <int>            # optional — default: 0 (no retry)
      timeout_seconds: <int>        # optional — default: 0 (no timeout)
      retry_delay_seconds: <int>    # optional — default: 5
      retry_backoff: linear | exponential  # optional — default: linear
      save_state: true | false      # optional — default: false`

const pipelineExample = `Example:
version: 1
project: my-service
repo:
  url: https://github.com/org/my-service.git
  base_branch: main
output:
  type: pr
  branch_prefix: feat/kai-
steps:
  - id: scaffold
    prompt: |
      Initialize the project with a Go module and basic HTTP server.
    validation:
      - exit_zero
    approval: optional
  - id: implement-api
    depends_on:
      - scaffold
    prompt: |
      Implement the REST API endpoints.
    policy:
      allowed_dirs:
        - ./internal
      max_retries: 2
      timeout_seconds: 300
    validation:
      - exit_zero
      - lint
    approval: optional`

type chatRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message"`
}

type chatResponse struct {
	ConversationID  string `json:"conversation_id"`
	Reply           string `json:"reply"`
	Spec            string `json:"spec"`
	SpecUpdated     bool   `json:"spec_updated"`
	SuggestContinue bool   `json:"suggest_continue"`
}

type generateRequest struct {
	ConversationID string `json:"conversation_id"`
}

type generateResponse struct {
	ConversationID string `json:"conversation_id"`
	YAML           string `json:"yaml"`
}

type yamlCheck struct {
	Project string     `yaml:"project"`
	Steps   []yamlStep `yaml:"steps"`
}

type yamlStep struct {
	ID     string `yaml:"id"`
	Prompt string `yaml:"prompt"`
}

func handleChat(deps *Deps, w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	var sess *chat.Session
	if req.ConversationID != "" {
		sess = deps.Sessions.Get(req.ConversationID)
		if sess == nil {
			writeError(w, http.StatusNotFound, "conversation %s not found", req.ConversationID)
			return
		}
	} else {
		sess = deps.Sessions.Create()
	}

	sess.Messages = append(sess.Messages, llm.Message{Role: "user", Content: req.Message})

	messages := buildChatPrompt(sess.Spec, sess.Messages)

	reply, usage, err := deps.LLMClient.ChatCompletion(messages)
	if err != nil {
		deps.Logger.Error("chat completion failed", sdkgokit.F("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "LLM call failed: %v", err)
		return
	}

	sess.Messages = append(sess.Messages, llm.Message{Role: "assistant", Content: reply})

	specUpdated := false
	newSpec := sess.Spec
	if idx := strings.Index(reply, "<spec>"); idx != -1 {
		endIdx := strings.Index(reply[idx:], "</spec>")
		if endIdx != -1 {
			newSpec = reply[idx+6 : idx+endIdx]
			newSpec = strings.TrimSpace(newSpec)
			sess.Spec = newSpec
			specUpdated = true
		}
	}

	cleanReply := stripSpecTags(reply)
	cleanReply = strings.ReplaceAll(cleanReply, "[SUGGEST_CONTINUE]", "")
	cleanReply = strings.TrimSpace(cleanReply)

	suggestContinue := strings.Contains(reply, "[SUGGEST_CONTINUE]")

	deps.Logger.Info("chat message processed",
		sdkgokit.F("conversation_id", sess.ID),
		sdkgokit.F("spec_updated", specUpdated),
		sdkgokit.F("suggest_continue", suggestContinue),
		sdkgokit.F("tokens", usage.TotalTokens),
	)

	resp := chatResponse{
		ConversationID:  sess.ID,
		Reply:           cleanReply,
		Spec:            sess.Spec,
		SpecUpdated:     specUpdated,
		SuggestContinue: suggestContinue,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleGeneratePipeline(deps *Deps, w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: %v", err)
		return
	}
	if req.ConversationID == "" {
		writeError(w, http.StatusBadRequest, "conversation_id is required")
		return
	}

	sess := deps.Sessions.Get(req.ConversationID)
	if sess == nil {
		writeError(w, http.StatusNotFound, "conversation %s not found", req.ConversationID)
		return
	}

	if sess.Spec == "" {
		writeError(w, http.StatusBadRequest, "no spec to generate pipeline from")
		return
	}

	systemContent := fmt.Sprintf(`You are a pipeline.yaml generator. Given a project specification, generate a complete pipeline.yaml file.

%s

%s

Return ONLY the YAML inside %s tags. No other text.`, pipelineSchema, pipelineExample, "```yaml```")

	messages := []llm.Message{
		{Role: "system", Content: systemContent},
		{Role: "user", Content: "Generate a pipeline.yaml from this specification:\n\n" + sess.Spec},
	}

	var yamlStr string
	for attempt := 0; attempt < 5; attempt++ {
		reply, usage, err := deps.LLMClient.ChatCompletion(messages)
		if err != nil {
			deps.Logger.Error("pipeline generation attempt failed", sdkgokit.F("attempt", attempt), sdkgokit.F("error", err.Error()))
			if attempt == 4 {
				writeError(w, http.StatusInternalServerError, "pipeline generation failed after 5 attempts: %v", err)
				return
			}
			messages = append(messages, llm.Message{Role: "assistant", Content: reply})
			messages = append(messages, llm.Message{Role: "user", Content: fmt.Sprintf("Error: %v. Please try again, returning ONLY valid YAML inside ```yaml ... ``` tags.", err)})
			continue
		}

		yamlStr = extractYAML(reply)
		if yamlStr == "" {
			deps.Logger.Warn("no yaml found in LLM response", sdkgokit.F("attempt", attempt))
			if attempt == 4 {
				writeError(w, http.StatusInternalServerError, "LLM did not return valid YAML after 5 attempts")
				return
			}
			messages = append(messages, llm.Message{Role: "assistant", Content: reply})
			messages = append(messages, llm.Message{Role: "user", Content: "The response did not contain YAML. Wrap the pipeline.yaml inside ```yaml ... ``` tags with no other text."})
			continue
		}

		if err := validateYAML(yamlStr); err != nil {
			deps.Logger.Warn("yaml validation failed", sdkgokit.F("attempt", attempt), sdkgokit.F("error", err.Error()))
			if attempt == 4 {
				writeError(w, http.StatusBadRequest, "generated pipeline YAML is invalid: %v", err)
				return
			}
			messages = append(messages, llm.Message{Role: "assistant", Content: reply})
			messages = append(messages, llm.Message{Role: "user", Content: fmt.Sprintf("The YAML has an error: %v. Please fix it and return corrected YAML inside ```yaml ... ``` tags.", err)})
			continue
		}

		deps.Logger.Info("pipeline generated",
			sdkgokit.F("conversation_id", sess.ID),
			sdkgokit.F("tokens", usage.TotalTokens),
			sdkgokit.F("attempts", attempt+1),
		)
		break
	}

	resp := generateResponse{
		ConversationID: sess.ID,
		YAML:           yamlStr,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func validateYAML(yamlStr string) error {
	var chk yamlCheck
	if err := yaml.Unmarshal([]byte(yamlStr), &chk); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}
	if chk.Project == "" {
		return fmt.Errorf("missing required field: project")
	}
	if len(chk.Steps) == 0 {
		return fmt.Errorf("pipeline must have at least one step")
	}
	for i, s := range chk.Steps {
		if s.ID == "" {
			return fmt.Errorf("step %d: missing required field: id", i)
		}
		if s.Prompt == "" {
			return fmt.Errorf("step %q: missing required field: prompt", s.ID)
		}
	}
	return nil
}

func buildChatPrompt(spec string, history []llm.Message) []llm.Message {
	systemPrompt := "You are a pipeline planning assistant. Your job is to build a detailed project specification document that will be used to generate a pipeline.yaml file.\n\n"
	systemPrompt += "You have the ability to edit the project spec document. ONLY edit the spec when the user requests a change or when you identify something important missing.\n"
	systemPrompt += "When you edit the spec, include the FULL updated spec in <spec>...</spec> tags.\n"
	systemPrompt += "Conversational replies go outside the tags.\n\n"
	systemPrompt += "The spec should be a plain-text document covering:\n"
	systemPrompt += "- Project name and purpose\n"
	systemPrompt += "- Repository URL and base branch\n"
	systemPrompt += "- Output type (pr / branch / direct) and branch prefix\n"
	systemPrompt += "- Every pipeline step with: step ID, detailed prompt, dependencies on other steps, validation gates (exit_zero, lint, typecheck, tests, diff_review), and approval requirement (optional / required)\n\n"
	systemPrompt += "You can also include policy constraints per step: allowed_dirs, allowed_tools, allowed_commands, agent, max_retries, timeout_seconds, retry_delay_seconds, retry_backoff (linear/exponential), save_state.\n\n"
	systemPrompt += "Refer to this schema when discussing the pipeline with the user:\n\n"
	systemPrompt += pipelineSchema + "\n\n"
	systemPrompt += "Structure the spec clearly with sections so it can be directly converted to YAML. Ask the user questions to fill in missing details. When all key points are covered, end your reply with [SUGGEST_CONTINUE]."

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
	}

	if spec != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: "Current project spec:\n" + spec,
		})
	}

	messages = append(messages, history...)
	return messages
}

func stripSpecTags(reply string) string {
	var result strings.Builder
	inTag := false
	i := 0
	for i < len(reply) {
		if strings.HasPrefix(reply[i:], "<spec>") {
			inTag = true
			i += 6
			continue
		}
		if inTag && strings.HasPrefix(reply[i:], "</spec>") {
			inTag = false
			i += 7
			continue
		}
		if !inTag {
			result.WriteByte(reply[i])
		}
		i++
	}
	return strings.TrimSpace(result.String())
}

func extractYAML(text string) string {
	start := strings.Index(text, "```yaml")
	if start == -1 {
		start = strings.Index(text, "```")
		if start == -1 {
			return ""
		}
		start += 3
	} else {
		start += 7
	}

	end := strings.Index(text[start:], "```")
	if end == -1 {
		return strings.TrimSpace(text[start:])
	}

	return strings.TrimSpace(text[start : start+end])
}
