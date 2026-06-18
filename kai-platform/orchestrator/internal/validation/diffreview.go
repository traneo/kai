package validation

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type DiffIssue struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type DiffReviewGate struct {
	maxFileSize   int64
	blockPatterns []*regexp.Regexp
	maxFiles      int
}

func NewDiffReviewGate() *DiffReviewGate {
	return &DiffReviewGate{
		maxFileSize: 1024 * 1024,
		blockPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:api[_-]?key|secret|password|token|credential)\s*[:=]\s*['"][^'"]+['"]`),
			regexp.MustCompile(`(?i)-----BEGIN (?:RSA |EC )?PRIVATE KEY-----`),
		},
		maxFiles: 50,
	}
}

func (g *DiffReviewGate) Name() Type {
	return TypeDiffReview
}

func (g *DiffReviewGate) Run(ctx *Context) *Result {
	issues, err := g.review(ctx.RepoDir)
	if err != nil {
		return &Result{
			Gate:    TypeDiffReview,
			Status:  StatusFailed,
			Message: fmt.Sprintf("diff review error: %v", err),
		}
	}

	if len(issues) == 0 {
		return &Result{
			Gate:    TypeDiffReview,
			Status:  StatusPassed,
			Message: "diff review passed, no issues found",
		}
	}

	var lines []string
	for _, iss := range issues {
		lines = append(lines, fmt.Sprintf("%s:%d: %s", iss.File, iss.Line, iss.Message))
	}

	return &Result{
		Gate:    TypeDiffReview,
		Status:  StatusFailed,
		Message: fmt.Sprintf("diff review found %d issues:\n%s", len(issues), strings.Join(lines, "\n")),
	}
}

func (g *DiffReviewGate) review(repoDir string) ([]DiffIssue, error) {
	diff, err := g.getDiff(repoDir)
	if err != nil {
		return nil, err
	}

	if len(diff) == 0 {
		return nil, nil
	}

	return g.reviewDiffContent(diff)
}

func (g *DiffReviewGate) getDiff(repoDir string) ([]byte, error) {
	cmd := exec.Command("git", "diff", "--diff-filter=AM")
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	if len(out) > 0 {
		return out, nil
	}

	cmd2 := exec.Command("git", "diff", "--cached", "--diff-filter=AM")
	if repoDir != "" {
		cmd2.Dir = repoDir
	}
	out2, err := cmd2.Output()
	if err != nil {
		return nil, nil
	}
	return out2, nil
}

func (g *DiffReviewGate) reviewDiffContent(diff []byte) ([]DiffIssue, error) {
	var issues []DiffIssue
	currentFile := ""
	lineNum := 0

	scanner := bufio.NewScanner(bytes.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			lineNum = 0
			continue
		}

		if strings.HasPrefix(line, "@@") {
			parts := strings.Split(line, " ")
			if len(parts) >= 3 {
				newRange := strings.Split(parts[2], ",")[0]
				var n int
				if _, err := fmt.Sscanf(newRange, "+%d", &n); err == nil && n > 0 {
					lineNum = n - 1
				}
			}
			continue
		}

		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			lineNum++
			content := strings.TrimPrefix(line, "+")

			for _, pattern := range g.blockPatterns {
				if pattern.MatchString(content) {
					issues = append(issues, DiffIssue{
						File:    currentFile,
						Line:    lineNum,
						Message: fmt.Sprintf("contains blocked pattern: %s", pattern.String()),
					})
					break
				}
			}
		}
	}

	if len(issues) > g.maxFiles {
		issues = issues[:g.maxFiles]
	}

	return issues, nil
}
