package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/plugins/llm"
	"github.com/flosch/pongo2/v6"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   i18n.T(i18n.MsgDiffCmd, nil),
	Short: i18n.T(i18n.MsgDiffCmdShort, nil),
	Args:  cobra.MinimumNArgs(2),
	RunE:  runDiff,
}

type DiffResult struct {
	Summary string
	Changes []Change
	Impact  string
}

type Change struct {
	Type         string
	Location     string
	Description  string
	Significance string
}

func runDiff(cmd *cobra.Command, args []string) error {
	oldContent := args[0]
	newContent := args[1]

	result, err := diffTexts(oldContent, newContent)
	if err != nil {
		return err
	}

	fmt.Print(i18n.T(i18n.MsgDiffSummary, pongo2.Context{"summary": result.Summary}))
	fmt.Print(i18n.T(i18n.MsgDiffImpact, pongo2.Context{"impact": result.Impact}))
	for _, c := range result.Changes {
		fmt.Print(i18n.T(i18n.MsgDiffChange, pongo2.Context{"type": c.Type, "location": c.Location, "description": c.Description}))
	}
	return nil
}

func diffTexts(oldContent, newContent string) (*DiffResult, error) {
	// Load LLM config
	llmConfigs, err := config.LoadLLMConfig("")
	if err != nil {
		return nil, fmt.Errorf("failed to load LLM config: %w", err)
	}

	if len(llmConfigs) == 0 {
		return nil, fmt.Errorf("LLM not configured: no providers found in config/llm.yaml or APP_PLUGINS_LLM")
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
		return nil, fmt.Errorf("LLM not configured: no valid providers available")
	}

	model := providerConfig.DefaultModel
	if model == "" {
		return nil, fmt.Errorf("model not configured: set default_model in LLM config")
	}

	// Build diff prompt
	prompt := buildDiffPrompt(oldContent, newContent)

	// Call LLM
	resp, err := provider.Invoke(context.Background(), prompt, model, 0.3)
	if err != nil {
		return nil, fmt.Errorf("LLM invocation failed: %w", err)
	}

	// Parse JSON response
	var result DiffResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// If JSON parsing fails, try to parse as text and create a simple result
		return parseTextDiffResponse(resp.Content, oldContent, newContent)
	}

	return &result, nil
}

func buildDiffPrompt(oldContent, newContent string) string {
	return fmt.Sprintf(`Analyze the semantic differences between the two texts below.
Respond ONLY with a valid JSON object in this exact format (no markdown, no code blocks):
{"summary": "brief one-sentence summary of the overall change", "impact": "high|medium|low impact assessment", "changes": [{"type": "added|removed|modified|clarified", "location": "which part/section is affected", "description": "what changed", "significance": "high|medium|low"}]}

Old text:
%s

New text:
%s
`, oldContent, newContent)
}

func parseTextDiffResponse(content string, oldContent, newContent string) (*DiffResult, error) {
	// Try to extract useful information from unstructured response
	content = cleanResponse(content)

	result := &DiffResult{
		Summary: "Differences detected (see details below)",
		Impact:  "medium",
		Changes: []Change{
			{
				Type:        "modified",
				Location:    "content",
				Description: content,
			},
		},
	}

	// Heuristic: if new is longer, likely added content
	if len(newContent) > len(oldContent) {
		result.Impact = "medium"
		result.Changes = append(result.Changes, Change{
			Type:        "added",
			Location:    "overall",
			Description: fmt.Sprintf("Content expanded by %d characters", len(newContent)-len(oldContent)),
		})
	} else if len(newContent) < len(oldContent) {
		result.Impact = "medium"
		result.Changes = append(result.Changes, Change{
			Type:        "removed",
			Location:    "overall",
			Description: fmt.Sprintf("Content reduced by %d characters", len(oldContent)-len(newContent)),
		})
	}

	return result, nil
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
