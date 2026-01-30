package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/jsonutil"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

// CodingAgent implements work tickets by invoking the agent to write or modify code.
// It builds a prompt from the ticket (ID, title, description, files to create/modify,
// acceptance criteria) and runs the agent in the project directory with context files.
type CodingAgent struct {
	caller     *Caller
	projectDir string
}

// NewCodingAgent creates a CodingAgent that uses the given Caller and project directory.
func NewCodingAgent(caller *Caller, projectDir string) *CodingAgent {
	return &CodingAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// Execute runs the agent to implement the given ticket. It builds a prompt from the ticket,
// collects context files from FilesToModify, and returns the agent Result and any error.
func (ca *CodingAgent) Execute(ctx context.Context, t *ticket.Ticket) (*Result, error) {
	prompt := ca.buildPrompt(t)

	// Collect context files
	contextFiles := make([]string, 0)
	for _, f := range t.FilesToModify {
		fullPath := filepath.Join(ca.projectDir, f)
		if _, err := os.Stat(fullPath); err == nil {
			contextFiles = append(contextFiles, fullPath)
		}
	}

	opts := []CallOption{
		WithWorkingDir(ca.projectDir),
		WithTimeout(10 * time.Minute),
	}

	if len(contextFiles) > 0 {
		opts = append(opts, WithContextFiles(contextFiles...))
	}

	return ca.caller.Call(ctx, prompt, opts...)
}

// buildPrompt creates the prompt for the coding agent
func (ca *CodingAgent) buildPrompt(t *ticket.Ticket) string {
	var sb strings.Builder

	sb.WriteString(i18n.AgentCodingIntro)
	sb.WriteString(fmt.Sprintf(i18n.AgentCodingProjectRoot, ca.projectDir))
	sb.WriteString(i18n.AgentCodingSectionTicket)
	sb.WriteString(fmt.Sprintf(i18n.AgentCodingTicketId, t.ID))
	sb.WriteString(fmt.Sprintf(i18n.AgentCodingTicketTitle, t.Title))
	sb.WriteString(fmt.Sprintf(i18n.AgentCodingTicketDesc, t.Description))
	sb.WriteString(fmt.Sprintf(i18n.AgentCodingTicketType, t.Type))
	sb.WriteString(fmt.Sprintf(i18n.AgentCodingTicketComplexity, t.EstimatedComplexity))

	if len(t.FilesToCreate) > 0 {
		sb.WriteString(i18n.AgentCodingSectionFilesCreate)
		for _, f := range t.FilesToCreate {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if len(t.FilesToModify) > 0 {
		sb.WriteString(i18n.AgentCodingSectionFilesModify)
		for _, f := range t.FilesToModify {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if len(t.AcceptanceCriteria) > 0 {
		sb.WriteString(i18n.AgentCodingSectionAcceptance)
		for _, c := range t.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(i18n.AgentCodingSteps)

	return sb.String()
}

// AnalyzeAgent analyzes existing project code and generates issues (performance, refactor, security, test, docs).
// It invokes the agent to produce a JSON report and parses it into ticket.IssueList.
type AnalyzeAgent struct {
	caller     *Caller
	projectDir string
}

// NewAnalyzeAgent creates an AnalyzeAgent that uses the given Caller and project directory.
func NewAnalyzeAgent(caller *Caller, projectDir string) *AnalyzeAgent {
	return &AnalyzeAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// AnalyzeScope defines which aspects of the codebase to analyze (performance, refactor, security, test, docs).
// Enable one or more flags to narrow or broaden the analysis.
type AnalyzeScope struct {
	Performance bool
	Refactor    bool
	Security    bool
	Test        bool
	Docs        bool
}

// AllScopes returns an AnalyzeScope with all analysis options enabled.
func AllScopes() AnalyzeScope {
	return AnalyzeScope{
		Performance: true,
		Refactor:    true,
		Security:    true,
		Test:        true,
		Docs:        true,
	}
}

// ParseScopes parses scope strings (e.g. "performance", "perf", "all") into an AnalyzeScope.
// "all" returns AllScopes(); other values map to Performance, Refactor, Security, Test, Docs.
func ParseScopes(scopes []string) AnalyzeScope {
	as := AnalyzeScope{}
	for _, s := range scopes {
		switch strings.ToLower(s) {
		case "all":
			return AllScopes()
		case "performance", "perf":
			as.Performance = true
		case "refactor":
			as.Refactor = true
		case "security", "sec":
			as.Security = true
		case "test":
			as.Test = true
		case "docs":
			as.Docs = true
		}
	}
	return as
}

// Analyze runs the agent to analyze the project according to scope and returns an IssueList.
// Output is written to .tickets/analysis-result.json and parsed into issues. On dry run, returns mock issues.
func (aa *AnalyzeAgent) Analyze(ctx context.Context, scope AnalyzeScope) (*ticket.IssueList, error) {
	prompt := aa.buildAnalyzePrompt(scope)

	outputFile := filepath.Join(aa.projectDir, ".tickets", "analysis-result.json")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return nil, fmt.Errorf(i18n.ErrAgentMkdirOutput, err)
	}

	result, jsonData, err := aa.caller.CallForJSON(ctx, prompt, outputFile,
		WithWorkingDir(aa.projectDir),
		WithTimeout(15*time.Minute),
	)

	if err != nil {
		if aa.caller.DryRun {
			return aa.createMockIssues(scope), nil
		}
		return nil, fmt.Errorf(i18n.ErrAgentAnalyzeFailed, err)
	}

	if !result.Success {
		return nil, fmt.Errorf(i18n.ErrAgentAnalyzeOutput, result.Error)
	}

	return aa.parseIssues(jsonData)
}

// buildAnalyzePrompt creates the prompt for analysis
func (aa *AnalyzeAgent) buildAnalyzePrompt(scope AnalyzeScope) string {
	var sb strings.Builder

	sb.WriteString(i18n.AgentAnalyzeIntro)
	sb.WriteString(fmt.Sprintf(i18n.AgentAnalyzeProjectDir, aa.projectDir))
	sb.WriteString(i18n.AgentAnalyzeAspects)
	if scope.Performance {
		sb.WriteString(i18n.AgentAnalyzePerf)
	}
	if scope.Refactor {
		sb.WriteString(i18n.AgentAnalyzeRefactor)
	}
	if scope.Security {
		sb.WriteString(i18n.AgentAnalyzeSecurity)
	}
	if scope.Test {
		sb.WriteString(i18n.AgentAnalyzeTest)
	}
	if scope.Docs {
		sb.WriteString(i18n.AgentAnalyzeDocs)
	}
	sb.WriteString(i18n.AgentAnalyzeJSONOutput)

	return sb.String()
}

// parseIssues parses the JSON output into issues
func (aa *AnalyzeAgent) parseIssues(data map[string]interface{}) (*ticket.IssueList, error) {
	issuesData, ok := data["issues"].([]interface{})
	if !ok {
		return nil, fmt.Errorf(i18n.ErrAgentInvalidIssues)
	}

	il := ticket.NewIssueList()
	for _, id := range issuesData {
		issueMap, ok := id.(map[string]interface{})
		if !ok {
			continue
		}

		issue := &ticket.Issue{
			ID:          jsonutil.GetString(issueMap, "id"),
			Category:    jsonutil.GetString(issueMap, "category"),
			Severity:    jsonutil.GetString(issueMap, "severity"),
			Title:       jsonutil.GetString(issueMap, "title"),
			Description: jsonutil.GetString(issueMap, "description"),
			Location:    jsonutil.GetString(issueMap, "location"),
			Suggestion:  jsonutil.GetString(issueMap, "suggestion"),
		}

		if issue.ID != "" && issue.Title != "" {
			il.Add(issue)
		}
	}

	return il, nil
}

// createMockIssues creates mock issues for dry run
func (aa *AnalyzeAgent) createMockIssues(scope AnalyzeScope) *ticket.IssueList {
	il := ticket.NewIssueList()

	if scope.Performance {
		il.Add(&ticket.Issue{
			ID:          "ISSUE-001",
			Category:    "performance",
			Severity:    "HIGH",
			Title:       "N+1 查詢問題",
			Description: "在迴圈中執行資料庫查詢",
			Location:    "service/user.go:45",
			Suggestion:  "使用批次查詢或 eager loading",
		})
	}

	if scope.Refactor {
		il.Add(&ticket.Issue{
			ID:          "ISSUE-002",
			Category:    "refactor",
			Severity:    "MED",
			Title:       "過長方法需拆分",
			Description: "方法超過 100 行，應該拆分成更小的函數",
			Location:    "handler/api.go:50-180",
			Suggestion:  "將方法拆分成多個小型函數",
		})
	}

	if scope.Security {
		il.Add(&ticket.Issue{
			ID:          "ISSUE-003",
			Category:    "security",
			Severity:    "HIGH",
			Title:       "硬編碼密碼",
			Description: "在程式碼中發現硬編碼的密碼",
			Location:    "config/db.go:15",
			Suggestion:  "使用環境變數或設定檔",
		})
	}

	return il
}
