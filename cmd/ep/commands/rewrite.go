package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/plugins/llm"
	"github.com/spf13/cobra"
)

var rewriteCmd = &cobra.Command{
	Use:   i18n.T(i18n.MsgRewriteCmd, nil),
	Short: i18n.T(i18n.MsgRewriteCmdShort, nil),
	Args:  cobra.MinimumNArgs(2),
	RunE:  runRewrite,
}

func runRewrite(cmd *cobra.Command, args []string) error {
	content := args[0]
	instruction := args[1]

	result, err := rewriteText(content, instruction)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

func rewriteText(content, instruction string) (string, error) {
	// Load LLM config
	llmConfigs, err := config.LoadLLMConfig("")
	if err != nil {
		return "", fmt.Errorf("failed to load LLM config: %w", err)
	}

	if len(llmConfigs) == 0 {
		return "", fmt.Errorf("LLM not configured: no providers found in config/llm.yaml or APP_PLUGINS_LLM")
	}

	// Find default provider or use first available
	var providerConfig config.LLMProviderConfig
	var provider llm.Interface
	found := false

	for _, cfg := range llmConfigs {
		if cfg.Default && cfg.Provider != "" {
			providerConfig = cfg
			provider, err = llm.NewProvider(cfg)
			if err == nil {
				found = true
				break
			}
		}
	}

	// If no default found, use first available
	if !found {
		for _, cfg := range llmConfigs {
			if cfg.Provider != "" {
				providerConfig = cfg
				provider, err = llm.NewProvider(cfg)
				if err == nil {
					found = true
					break
				}
			}
		}
	}

	if !found || provider == nil {
		return "", fmt.Errorf("LLM not configured: no valid providers available")
	}

	model := providerConfig.DefaultModel
	if model == "" {
		return "", fmt.Errorf("model not configured: set default_model in LLM config")
	}

	// Build rewrite prompt
	prompt := buildRewritePrompt(content, instruction)

	// Call LLM
	resp, err := provider.Invoke(context.Background(), prompt, model, 0.3)
	if err != nil {
		return "", fmt.Errorf("LLM invocation failed: %w", err)
	}

	// Clean response - remove markdown formatting
	return cleanResponse(resp.Content), nil
}

func buildRewritePrompt(content, instruction string) string {
	return fmt.Sprintf(`Rewrite the following text according to the instruction.
Do not include any markdown formatting, no code blocks, no think tags.
Just output the rewritten text directly.

Instruction: %s

Original text:
%s

Rewritten text:
`, instruction, content)
}

func cleanResponse(content string) string {
	// Remove code block markers
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimPrefix(content, "```markdown")
	content = strings.TrimPrefix(content, "```text")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSuffix(content, "```")

	// Remove bold/italic markers
	content = strings.ReplaceAll(content, "**", "")
	content = strings.ReplaceAll(content, "*", "")
	content = strings.ReplaceAll(content, "_", "")

	return strings.TrimSpace(content)
}

func init() {
	rootCmd.AddCommand(rewriteCmd)
}
