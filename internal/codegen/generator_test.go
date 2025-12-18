package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datagendev/datagen-cli/internal/config"
)

func TestGenerateProject_WebhookSignatureVerificationChecksSecretBeforeHeader(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()
	cfg := &config.DatagenConfig{
		DatagenAPIKeyEnv: "DATAGEN_API_KEY",
		ClaudeAPIKeyEnv:  "ANTHROPIC_API_KEY",
		Services: []config.Service{
			{
				Name:        "poem_writer",
				Type:        "webhook",
				Description: "Poem writer webhook",
				Prompt:      ".claude/agents/poem-writer.md",
				WebhookPath: "/webhook/poem_writer",
				InputSchema: config.Schema{Fields: []config.Field{}},
				Webhook: &config.WebhookConfig{
					SignatureVerification: "hmac_sha256",
					SignatureHeader:       "X-Signature",
					SecretEnv:             "POEM_WRITER_HMAC_SECRET",
					RetryEnabled:          false,
				},
			},
		},
	}

	if err := GenerateProject(cfg, outDir); err != nil {
		t.Fatalf("GenerateProject: %v", err)
	}

	mainPath := filepath.Join(outDir, "app", "main.py")
	data, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.py: %v", err)
	}
	src := string(data)

	// Signature verification is optional when the secret is not configured.
	// Ensure the generated code checks for the secret before requiring the signature header.
	secretLine := `secret = getattr(settings, "poem_writer_hmac_secret", None)`
	signatureLine := `signature = request.headers.get("X-Signature")`

	secretIdx := strings.Index(src, secretLine)
	if secretIdx == -1 {
		t.Fatalf("expected to find %q in main.py", secretLine)
	}
	signatureIdx := strings.Index(src, signatureLine)
	if signatureIdx == -1 {
		t.Fatalf("expected to find %q in main.py", signatureLine)
	}
	if signatureIdx < secretIdx {
		t.Fatalf("expected secret check before signature header check; secretIdx=%d signatureIdx=%d", secretIdx, signatureIdx)
	}
}

func TestGenerateProject_WebhookNoSignatureVerificationOmitsSignatureHelper(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()
	cfg := &config.DatagenConfig{
		DatagenAPIKeyEnv: "DATAGEN_API_KEY",
		ClaudeAPIKeyEnv:  "ANTHROPIC_API_KEY",
		Services: []config.Service{
			{
				Name:        "poem_writer",
				Type:        "webhook",
				Description: "Poem writer webhook",
				Prompt:      ".claude/agents/poem-writer.md",
				WebhookPath: "/webhook/poem_writer",
				InputSchema: config.Schema{Fields: []config.Field{}},
				Webhook: &config.WebhookConfig{
					SignatureVerification: "none",
					RetryEnabled:          false,
				},
			},
		},
	}

	if err := GenerateProject(cfg, outDir); err != nil {
		t.Fatalf("GenerateProject: %v", err)
	}

	mainPath := filepath.Join(outDir, "app", "main.py")
	data, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.py: %v", err)
	}
	src := string(data)

	if strings.Contains(src, "verify_poem_writer_signature") {
		t.Fatalf("did not expect signature verification helper to be generated when signature_verification=none")
	}
}
