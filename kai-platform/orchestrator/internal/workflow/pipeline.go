package workflow

import "time"

type Pipeline struct {
	Version int      `yaml:"version"`
	Project string   `yaml:"project"`
	Repo    Repo     `yaml:"repo"`
	Output  Output   `yaml:"output"`
	Steps   []Step   `yaml:"steps"`
}

type Repo struct {
	URL        string `yaml:"url"`
	BaseBranch string `yaml:"base_branch"`
	Provider   string `yaml:"provider"`
	TokenRef   string `yaml:"token_ref"`
}

type Output struct {
	Type         string `yaml:"type"`
	BranchPrefix string `yaml:"branch_prefix"`
}

type GateResult struct {
	Gate     string `json:"gate"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Duration string `json:"duration"`
}

type Step struct {
	ID         string   `yaml:"id"`
	Prompt     string   `yaml:"prompt"`
	DependsOn  []string `yaml:"depends_on"`
	Policy     Policy   `yaml:"policy"`
	Validation []string `yaml:"validation"`
	Approval   string   `yaml:"approval"`
}

type Policy struct {
	AllowedDirs      []string `yaml:"allowed_dirs"`
	AllowedTools     []string `yaml:"allowed_tools"`
	AllowedCommands  []string `yaml:"allowed_commands"`
	Agent            string   `yaml:"agent"`
	MaxRetries       int      `yaml:"max_retries"`
	RetryDelaySeconds int     `yaml:"retry_delay_seconds"`
	RetryBackoff     string   `yaml:"retry_backoff"`
	TimeoutSeconds   int      `yaml:"timeout_seconds"`
	SaveState        bool     `yaml:"save_state"`
}

func (p Policy) RetryDelay(retryCount int) time.Duration {
	delay := p.RetryDelaySeconds
	if delay <= 0 {
		delay = 5
	}

	switch p.RetryBackoff {
	case "exponential":
		d := time.Duration(delay) * time.Second
		for i := 0; i < retryCount; i++ {
			d *= 2
		}
		if d > 5*time.Minute {
			d = 5 * time.Minute
		}
		return d
	default:
		return time.Duration(delay) * time.Second
	}
}

func (s Step) RequiresApproval() bool {
	return s.Approval == "required"
}

type ValidationGate string

const (
	GateExitCode    ValidationGate = "exit_zero"
	GateLint        ValidationGate = "lint"
	GateTypecheck   ValidationGate = "typecheck"
	GateTests       ValidationGate = "tests"
	GateDiffReview  ValidationGate = "diff_review"
)
