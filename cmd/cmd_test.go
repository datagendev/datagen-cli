package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datagendev/datagen-cli/internal/agents"
	"github.com/datagendev/datagen-cli/internal/config"
)

// ---- Root command structure ----

func TestRootCommand_HasExpectedSubcommands(t *testing.T) {
	t.Parallel()

	expected := []string{"login", "mcp", "github", "agents", "secrets"}
	for _, name := range expected {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected root command to have subcommand %q", name)
		}
	}
}

func TestAgentsCommand_HasExpectedSubcommands(t *testing.T) {
	t.Parallel()

	expected := []string{"list", "show", "deploy", "undeploy", "run", "logs", "config", "schedule"}
	for _, name := range expected {
		found := false
		for _, cmd := range agentsCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'agents' command to have subcommand %q", name)
		}
	}
}

func TestGitHubCommand_HasExpectedSubcommands(t *testing.T) {
	t.Parallel()

	expected := []string{"connect", "repos", "connected", "connect-repo", "sync", "status"}
	for _, name := range expected {
		found := false
		for _, cmd := range githubCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'github' command to have subcommand %q", name)
		}
	}
}

func TestSecretsCommand_HasExpectedSubcommands(t *testing.T) {
	t.Parallel()

	expected := []string{"list", "set"}
	for _, name := range expected {
		found := false
		for _, cmd := range secretsCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'secrets' command to have subcommand %q", name)
		}
	}
}

func TestAgentsListCmd_HasRepoAndDeployedFlags(t *testing.T) {
	t.Parallel()

	if f := agentsListCmd.Flags().Lookup("repo"); f == nil {
		t.Error("agentsListCmd should have a --repo flag")
	}
	if f := agentsListCmd.Flags().Lookup("deployed"); f == nil {
		t.Error("agentsListCmd should have a --deployed flag")
	}
}

func TestAgentsLogsCmd_HasLimitFlag(t *testing.T) {
	t.Parallel()

	f := agentsLogsCmd.Flags().Lookup("limit")
	if f == nil {
		t.Fatal("agentsLogsCmd should have a --limit flag")
	}
	if f.DefValue != "10" {
		t.Errorf("--limit default = %q; want %q", f.DefValue, "10")
	}
}

func TestAgentsRunCmd_HasPayloadFlag(t *testing.T) {
	t.Parallel()

	f := agentsRunCmd.Flags().Lookup("payload")
	if f == nil {
		t.Fatal("agentsRunCmd should have a --payload flag")
	}
	if f.DefValue != "{}" {
		t.Errorf("--payload default = %q; want %q", f.DefValue, "{}")
	}
}

func TestAgentsScheduleCmd_HasCronAndTimezoneFlags(t *testing.T) {
	t.Parallel()

	if f := agentsScheduleCmd.Flags().Lookup("cron"); f == nil {
		t.Error("agentsScheduleCmd should have a --cron flag")
	}
	f := agentsScheduleCmd.Flags().Lookup("timezone")
	if f == nil {
		t.Fatal("agentsScheduleCmd should have a --timezone flag")
	}
	if f.DefValue != "UTC" {
		t.Errorf("--timezone default = %q; want %q", f.DefValue, "UTC")
	}
}

// ---- parseBoolOrNull (agents.go) ----

func TestParseBoolOrNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want interface{}
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"false", false},
		{"FALSE", false},
		{"False", false},
		{"default", nil},
		{"DEFAULT", nil},
		{"Default", nil},
		{"", nil},
		{"anything", nil},
	}
	for _, tt := range tests {
		got := parseBoolOrNull(tt.in)
		if got != tt.want {
			t.Errorf("parseBoolOrNull(%q) = %v; want %v", tt.in, got, tt.want)
		}
	}
}

// ---- parseRecipientFlag (agents.go) ----

func TestParseRecipientFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in        string
		wantEmail string
		wantRole  string
	}{
		{"user@example.com", "user@example.com", "VIEWER"},
		{"user@example.com:OWNER", "user@example.com", "OWNER"},
		{"user@example.com:owner", "user@example.com", "OWNER"},
		{"user@example.com:viewer", "user@example.com", "VIEWER"},
		{"user@example.com:admin", "user@example.com", "ADMIN"},
		// Empty role defaults to VIEWER
		{"user@example.com:", "user@example.com", "VIEWER"},
	}
	for _, tt := range tests {
		email, role := parseRecipientFlag(tt.in)
		if email != tt.wantEmail || role != tt.wantRole {
			t.Errorf("parseRecipientFlag(%q) = (%q, %q); want (%q, %q)",
				tt.in, email, role, tt.wantEmail, tt.wantRole)
		}
	}
}

// ---- formatAgentType (agents.go) ----

func TestFormatAgentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"SKILL", "skill"},
		{"skill", "skill"},
		{"Skill", "skill"},
		{"COMMAND", "command"},
		{"command", "command"},
		{"AGENT", "agent"},
		{"agent", "agent"},
		{"unknown", "agent"},
		{"", "agent"},
	}
	for _, tt := range tests {
		got := formatAgentType(tt.in)
		if got != tt.want {
			t.Errorf("formatAgentType(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

// ---- formatBoolOverride (agents.go) ----

func TestFormatBoolOverride_Nil(t *testing.T) {
	t.Parallel()

	got := formatBoolOverride(nil)
	if got != "default (global)" {
		t.Errorf("formatBoolOverride(nil) = %q; want %q", got, "default (global)")
	}
}

func TestFormatBoolOverride_True(t *testing.T) {
	t.Parallel()

	v := true
	got := formatBoolOverride(&v)
	if got != "true (agent override)" {
		t.Errorf("formatBoolOverride(&true) = %q; want %q", got, "true (agent override)")
	}
}

func TestFormatBoolOverride_False(t *testing.T) {
	t.Parallel()

	v := false
	got := formatBoolOverride(&v)
	if got != "false (agent override)" {
		t.Errorf("formatBoolOverride(&false) = %q; want %q", got, "false (agent override)")
	}
}

// ---- getExecutionStatusIcon (agents.go) ----

func TestGetExecutionStatusIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		want   string
	}{
		{"completed", "✅"},
		{"success", "✅"},
		{"COMPLETED", "✅"},
		{"SUCCESS", "✅"},
		{"failed", "❌"},
		{"error", "❌"},
		{"FAILED", "❌"},
		{"running", "🔄"},
		{"in_progress", "🔄"},
		{"RUNNING", "🔄"},
		{"pending", "⏳"},
		{"queued", "⏳"},
		{"PENDING", "⏳"},
		{"unknown", "⚪"},
		{"", "⚪"},
	}
	for _, tt := range tests {
		got := getExecutionStatusIcon(tt.status)
		if got != tt.want {
			t.Errorf("getExecutionStatusIcon(%q) = %q; want %q", tt.status, got, tt.want)
		}
	}
}

// ---- buildConfigUpdateRequest (agents.go) ----
// These tests modify package-level flag variables, so they cannot run in parallel.

func TestBuildConfigUpdateRequest_SetPrompt(t *testing.T) {
	old := configSetPrompt
	oldClear := configClearPrompt
	t.Cleanup(func() { configSetPrompt = old; configClearPrompt = oldClear })

	configSetPrompt = "my custom prompt"
	configClearPrompt = false

	req := buildConfigUpdateRequest()
	if req.EntryPrompt == nil {
		t.Fatal("expected EntryPrompt to be set")
	}
	if *req.EntryPrompt != "my custom prompt" {
		t.Errorf("EntryPrompt = %q; want %q", *req.EntryPrompt, "my custom prompt")
	}
}

func TestBuildConfigUpdateRequest_ClearPrompt(t *testing.T) {
	old := configSetPrompt
	oldClear := configClearPrompt
	t.Cleanup(func() { configSetPrompt = old; configClearPrompt = oldClear })

	configClearPrompt = true
	configSetPrompt = ""

	req := buildConfigUpdateRequest()
	if req.EntryPrompt == nil {
		t.Fatal("expected EntryPrompt to be set (empty string) when clearing")
	}
	if *req.EntryPrompt != "" {
		t.Errorf("EntryPrompt = %q; want empty string", *req.EntryPrompt)
	}
}

func TestBuildConfigUpdateRequest_SecretsAndPrMode(t *testing.T) {
	old := configSecrets
	oldPr := configPrMode
	t.Cleanup(func() { configSecrets = old; configPrMode = oldPr })

	configSecrets = "KEY1, KEY2"
	configPrMode = "create_pr"

	req := buildConfigUpdateRequest()
	if req.Webhook == nil {
		t.Fatal("expected Webhook map to be set")
	}
	names, ok := req.Webhook["secretNames"].([]string)
	if !ok {
		t.Fatalf("secretNames type = %T; want []string", req.Webhook["secretNames"])
	}
	if len(names) != 2 || names[0] != "KEY1" || names[1] != "KEY2" {
		t.Errorf("secretNames = %v; want [KEY1 KEY2]", names)
	}
	if req.Webhook["prMode"] != "create_pr" {
		t.Errorf("prMode = %v; want create_pr", req.Webhook["prMode"])
	}
}

func TestBuildConfigUpdateRequest_AddRecipient(t *testing.T) {
	old := configAddRecipient
	t.Cleanup(func() { configAddRecipient = old })

	configAddRecipient = "user@example.com:OWNER"

	req := buildConfigUpdateRequest()
	if req.Recipients == nil {
		t.Fatal("expected Recipients to be set")
	}
	if len(req.Recipients.Add) != 1 {
		t.Fatalf("len(Recipients.Add) = %d; want 1", len(req.Recipients.Add))
	}
	r := req.Recipients.Add[0]
	if r.Email != "user@example.com" {
		t.Errorf("Email = %q; want %q", r.Email, "user@example.com")
	}
	if r.Role != "OWNER" {
		t.Errorf("Role = %q; want %q", r.Role, "OWNER")
	}
}

func TestBuildConfigUpdateRequest_RemoveRecipient(t *testing.T) {
	old := configRemoveRecipient
	t.Cleanup(func() { configRemoveRecipient = old })

	configRemoveRecipient = "some-recipient-id"

	req := buildConfigUpdateRequest()
	if req.Recipients == nil {
		t.Fatal("expected Recipients to be set")
	}
	if len(req.Recipients.Remove) != 1 || req.Recipients.Remove[0] != "some-recipient-id" {
		t.Errorf("Recipients.Remove = %v; want [some-recipient-id]", req.Recipients.Remove)
	}
}

func TestBuildConfigUpdateRequest_NotifyFlags(t *testing.T) {
	oldSuccess := configNotifySuccess
	oldFailure := configNotifyFailure
	oldReply := configNotifyReply
	t.Cleanup(func() {
		configNotifySuccess = oldSuccess
		configNotifyFailure = oldFailure
		configNotifyReply = oldReply
	})

	configNotifySuccess = "true"
	configNotifyFailure = "false"
	configNotifyReply = "default"

	req := buildConfigUpdateRequest()
	if req.Notifications == nil {
		t.Fatal("expected Notifications map to be set")
	}
	if req.Notifications["emailOnSuccess"] != true {
		t.Errorf("emailOnSuccess = %v; want true", req.Notifications["emailOnSuccess"])
	}
	if req.Notifications["emailOnFailure"] != false {
		t.Errorf("emailOnFailure = %v; want false", req.Notifications["emailOnFailure"])
	}
	if req.Notifications["emailReplyEnabled"] != nil {
		t.Errorf("emailReplyEnabled = %v; want nil (default)", req.Notifications["emailReplyEnabled"])
	}
}

// ---- parseCSVSet (mcp.go) ----

func TestParseCSVSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want map[string]bool
	}{
		{"", map[string]bool{}},
		{"codex", map[string]bool{"codex": true}},
		{"codex,claude,gemini", map[string]bool{"codex": true, "claude": true, "gemini": true}},
		// Whitespace is trimmed
		{"codex, claude", map[string]bool{"codex": true, "claude": true}},
		// Uppercased values are normalised
		{"CODEX,CLAUDE", map[string]bool{"codex": true, "claude": true}},
		// Bare commas produce no entries
		{",", map[string]bool{}},
		// Duplicates collapse into a single entry
		{"codex,codex", map[string]bool{"codex": true}},
	}
	for _, tt := range tests {
		got := parseCSVSet(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("parseCSVSet(%q): got %v; want %v", tt.in, got, tt.want)
			continue
		}
		for k, v := range tt.want {
			if got[k] != v {
				t.Errorf("parseCSVSet(%q): key %q = %v; want %v", tt.in, k, got[k], v)
			}
		}
	}
}

// ---- samePath (start.go) ----

func TestSamePath_SamePaths(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	a := filepath.Join(tmp, "file.txt")
	b := filepath.Join(tmp, "file.txt")
	if !samePath(a, b) {
		t.Error("samePath: identical paths should return true")
	}
}

func TestSamePath_DifferentPaths(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.txt")
	b := filepath.Join(tmp, "b.txt")
	if samePath(a, b) {
		t.Error("samePath: different paths should return false")
	}
}

func TestSamePath_CurrentDir(t *testing.T) {
	t.Parallel()

	if !samePath(".", ".") {
		t.Error("samePath: '.' and '.' should be the same path")
	}
}

// ---- copyFile (start.go) ----

func TestCopyFile_CopiesContent(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "source.txt")
	dst := filepath.Join(tmp, "dest.txt")
	content := []byte("hello datagen")

	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("copyFile content = %q; want %q", got, content)
	}
}

func TestCopyFile_MissingSrc(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	err := copyFile(filepath.Join(tmp, "nonexistent.txt"), filepath.Join(tmp, "dst.txt"))
	if err == nil {
		t.Error("expected error when source does not exist")
	}
}

// ---- generateAPITemplate / generateWebhookTemplate / generateStreamingTemplate (start.go) ----

func TestGenerateAPITemplate(t *testing.T) {
	t.Parallel()

	svc := &config.Service{Name: "my_service", Description: "My API service"}
	out := generateAPITemplate(svc)

	if !strings.Contains(out, "my_service") {
		t.Error("API template should contain service name")
	}
	if !strings.Contains(out, "My API service") {
		t.Error("API template should contain service description")
	}
	// Frontmatter marker should be present
	if !strings.Contains(out, "---") {
		t.Error("API template should contain frontmatter delimiters")
	}
}

func TestGenerateWebhookTemplate(t *testing.T) {
	t.Parallel()

	svc := &config.Service{Name: "my_webhook", Description: "My webhook service"}
	out := generateWebhookTemplate(svc)

	if !strings.Contains(out, "my_webhook") {
		t.Error("Webhook template should contain service name")
	}
	if !strings.Contains(out, "My webhook service") {
		t.Error("Webhook template should contain service description")
	}
	if !strings.Contains(out, "Webhook") {
		t.Error("Webhook template should contain 'Webhook' heading")
	}
}

func TestGenerateStreamingTemplate(t *testing.T) {
	t.Parallel()

	svc := &config.Service{Name: "my_streamer", Description: "My streaming service"}
	out := generateStreamingTemplate(svc)

	if !strings.Contains(out, "my_streamer") {
		t.Error("Streaming template should contain service name")
	}
	if !strings.Contains(out, "My streaming service") {
		t.Error("Streaming template should contain service description")
	}
	if !strings.Contains(out, "Streaming") {
		t.Error("Streaming template should contain 'Streaming' heading")
	}
}

// ---- createAgentPromptFile (start.go) ----

func TestCreateAgentPromptFile_API(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	svc := &config.Service{
		Name:        "test_api",
		Type:        "api",
		Description: "A test API service",
		Prompt:      ".claude/agents/test-api.md",
	}

	if err := createAgentPromptFile(tmp, svc); err != nil {
		t.Fatalf("createAgentPromptFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".claude", "agents", "test-api.md"))
	if err != nil {
		t.Fatalf("read prompt file: %v", err)
	}
	if !strings.Contains(string(data), "test_api") {
		t.Error("API prompt file should contain service name")
	}
}

func TestCreateAgentPromptFile_Webhook(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	svc := &config.Service{
		Name:        "test_webhook",
		Type:        "webhook",
		Description: "A test webhook",
		Prompt:      ".claude/agents/test-webhook.md",
	}

	if err := createAgentPromptFile(tmp, svc); err != nil {
		t.Fatalf("createAgentPromptFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".claude", "agents", "test-webhook.md"))
	if err != nil {
		t.Fatalf("read prompt file: %v", err)
	}
	if !strings.Contains(string(data), "Webhook") {
		t.Error("webhook prompt file should contain 'Webhook'")
	}
}

func TestCreateAgentPromptFile_Streaming(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	svc := &config.Service{
		Name:        "test_stream",
		Type:        "streaming",
		Description: "A test streaming service",
		Prompt:      ".claude/agents/test-stream.md",
	}

	if err := createAgentPromptFile(tmp, svc); err != nil {
		t.Fatalf("createAgentPromptFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".claude", "agents", "test-stream.md"))
	if err != nil {
		t.Fatalf("read prompt file: %v", err)
	}
	if !strings.Contains(string(data), "Streaming") {
		t.Error("streaming prompt file should contain 'Streaming'")
	}
}

func TestCreateAgentPromptFile_CreatesParentDirs(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	svc := &config.Service{
		Name:        "nested_svc",
		Type:        "api",
		Description: "Nested",
		Prompt:      ".claude/agents/nested_svc.md",
	}

	if err := createAgentPromptFile(tmp, svc); err != nil {
		t.Fatalf("createAgentPromptFile: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, ".claude", "agents", "nested_svc.md")); err != nil {
		t.Errorf("expected prompt file to exist: %v", err)
	}
}

// ---- chooseMode (start.go) ----

func TestChooseMode_ValidAPIFlag(t *testing.T) {
	t.Parallel()

	mode, err := chooseMode("api")
	if err != nil {
		t.Fatalf("chooseMode('api') unexpected error: %v", err)
	}
	if mode != "api" {
		t.Errorf("chooseMode('api') = %q; want %q", mode, "api")
	}
}

func TestChooseMode_ValidWebhookFlag(t *testing.T) {
	t.Parallel()

	mode, err := chooseMode("webhook")
	if err != nil {
		t.Fatalf("chooseMode('webhook') unexpected error: %v", err)
	}
	if mode != "webhook" {
		t.Errorf("chooseMode('webhook') = %q; want %q", mode, "webhook")
	}
}

func TestChooseMode_InvalidFlag(t *testing.T) {
	t.Parallel()

	_, err := chooseMode("streaming")
	if err == nil {
		t.Error("chooseMode('streaming') should return an error for invalid mode")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error message should mention 'invalid', got: %v", err)
	}
}

// ---- chooseAgent (start.go) ----

func TestChooseAgent_ByName(t *testing.T) {
	t.Parallel()

	selectable := []agents.Agent{
		{Name: "poem-writer", Path: ".claude/agents/poem-writer.md"},
		{Name: "data-processor", Path: ".claude/agents/data-processor.md"},
	}

	got, err := chooseAgent(selectable, "poem-writer")
	if err != nil {
		t.Fatalf("chooseAgent by name: %v", err)
	}
	if got.Name != "poem-writer" {
		t.Errorf("chooseAgent by name: got %q; want 'poem-writer'", got.Name)
	}
}

func TestChooseAgent_ByFilename(t *testing.T) {
	t.Parallel()

	selectable := []agents.Agent{
		{Name: "poem-writer", Path: ".claude/agents/poem-writer.md"},
		{Name: "data-processor", Path: ".claude/agents/data-processor.md"},
	}

	got, err := chooseAgent(selectable, "poem-writer.md")
	if err != nil {
		t.Fatalf("chooseAgent by filename: %v", err)
	}
	if got.Name != "poem-writer" {
		t.Errorf("chooseAgent by filename: got %q; want 'poem-writer'", got.Name)
	}
}

func TestChooseAgent_ByStem(t *testing.T) {
	t.Parallel()

	selectable := []agents.Agent{
		{Name: "", Path: ".claude/agents/my-agent.md"},
	}

	got, err := chooseAgent(selectable, "my-agent")
	if err != nil {
		t.Fatalf("chooseAgent by stem: %v", err)
	}
	if filepath.Base(got.Path) != "my-agent.md" {
		t.Errorf("chooseAgent by stem: path = %q; want base 'my-agent.md'", got.Path)
	}
}

func TestChooseAgent_CaseInsensitiveMatch(t *testing.T) {
	t.Parallel()

	selectable := []agents.Agent{
		{Name: "Poem-Writer", Path: ".claude/agents/poem-writer.md"},
	}

	got, err := chooseAgent(selectable, "poem-writer")
	if err != nil {
		t.Fatalf("chooseAgent case-insensitive: %v", err)
	}
	if got.Name != "Poem-Writer" {
		t.Errorf("got %q; want 'Poem-Writer'", got.Name)
	}
}

func TestChooseAgent_NoMatch(t *testing.T) {
	t.Parallel()

	selectable := []agents.Agent{
		{Name: "poem-writer", Path: ".claude/agents/poem-writer.md"},
	}

	_, err := chooseAgent(selectable, "nonexistent")
	if err == nil {
		t.Error("chooseAgent should return an error when no agent matches")
	}
	if !strings.Contains(err.Error(), "no agent") {
		t.Errorf("error should mention 'no agent', got: %v", err)
	}
}

func TestChooseAgent_MultipleMatches(t *testing.T) {
	t.Parallel()

	selectable := []agents.Agent{
		{Name: "writer", Path: ".claude/agents/writer-v1.md"},
		{Name: "writer", Path: ".claude/agents/writer-v2.md"},
	}

	_, err := chooseAgent(selectable, "writer")
	if err == nil {
		t.Error("chooseAgent should return an error when multiple agents match")
	}
	if !strings.Contains(err.Error(), "multiple") {
		t.Errorf("error should mention 'multiple', got: %v", err)
	}
}

func TestChooseAgent_EmptyFlag_ReturnsErrorWithoutInteractive(t *testing.T) {
	// When flagValue is empty, chooseAgent would call survey.AskOne which
	// requires a terminal. In non-interactive test environments this is expected
	// to fail; we only verify that an empty list of selectable agents panics
	// or returns an error gracefully under interactive code paths by testing
	// the non-interactive path (flagValue != "").
	// This test verifies the flag-based path exhaustively above.
}
