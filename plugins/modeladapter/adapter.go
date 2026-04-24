// Package modeladapter provides prompt adaptation for different LLM models.
package modeladapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/eval-prompt/internal/service"
)

// Config holds model adapter configuration.
type Config struct {
	Enabled        bool
	RulesPath      string
	DefaultSource  string
	AutoLearn      bool
}

// Adapter implements the service.ModelAdapter interface.
type Adapter struct {
	rules   *RuleLibrary
	profiles map[string]service.ModelProfile
}

// NewAdapter creates a new ModelAdapter.
func NewAdapter(cfg Config) *Adapter {
	a := &Adapter{
		rules:    NewRuleLibrary(),
		profiles: make(map[string]service.ModelProfile),
	}

	// Register built-in model profiles
	a.registerBuiltInProfiles()

	return a
}

// Ensure Adapter implements service.ModelAdapter.
var _ service.ModelAdapter = (*Adapter)(nil)

// Adapt converts a prompt from source model to target model format.
func (a *Adapter) Adapt(ctx context.Context, prompt service.PromptContent, sourceModel, targetModel string) (service.AdaptedPrompt, error) {
	if sourceModel == targetModel {
		return service.AdaptedPrompt{Content: prompt.Instruction}, nil
	}

	result := service.AdaptedPrompt{
		ParamAdjustments: make(map[string]float64),
		FormatChanges:   []string{},
		Warnings:        []string{},
	}

	// Get source and target profiles
	sourceProfile, err := a.GetModelProfile(ctx, sourceModel)
	if err != nil {
		sourceProfile = a.getDefaultProfile(sourceModel)
	}

	targetProfile, err := a.GetModelProfile(ctx, targetModel)
	if err != nil {
		targetProfile = a.getDefaultProfile(targetModel)
	}

	// Apply format conversion rules
	result.Content = a.convertFormat(prompt, sourceProfile, targetProfile)

	// Apply parameter adjustments
	a.applyParamAdjustments(sourceProfile, targetProfile, result.ParamAdjustments)

	// Check context length
	a.checkContextLength(prompt, targetProfile, &result.Warnings)

	// Get adaptation rules
	rules := a.rules.GetRules(sourceModel, targetModel)
	for _, rule := range rules {
		result.Content = rule.Apply(result.Content)
		result.FormatChanges = append(result.FormatChanges, rule.Description())
	}

	return result, nil
}

// RecommendParams returns recommended parameters for a target model and task type.
func (a *Adapter) RecommendParams(ctx context.Context, targetModel string, taskType string) (service.ModelParams, error) {
	profile, err := a.GetModelProfile(ctx, targetModel)
	if err != nil {
		profile = a.getDefaultProfile(targetModel)
	}

	params := service.ModelParams{
		Temperature: 0.7,
		MaxTokens:    2048,
		TopP:        0.9,
	}

	// Adjust based on task type
	switch taskType {
	case "code_generation":
		params.Temperature = 0.3
		params.MaxTokens = 4096
	case "creative_writing":
		params.Temperature = 0.8
		params.MaxTokens = 2048
	case "analysis":
		params.Temperature = 0.5
		params.MaxTokens = 2048
	case "summarization":
		params.Temperature = 0.4
		params.MaxTokens = 512
	case "json_output":
		params.Temperature = 0.2
		params.MaxTokens = 2048
		if profile.JSONReliability > 0 {
			params.Temperature = 0.1 // Lower temperature for more consistent JSON
		}
	}

	// Adjust based on model profile
	switch profile.TemperatureCurve {
	case "steep":
		params.Temperature *= 0.8
	case "flat":
		params.Temperature *= 1.1
	}

	return params, nil
}

// EstimateScore estimates the expected score for a prompt on a target model.
func (a *Adapter) EstimateScore(ctx context.Context, promptID string, targetModel string) (float64, error) {
	// TODO: Implement based on historical adaptation data
	// For now, return a default estimate
	return 0.85, nil
}

// GetModelProfile returns the characteristics of a model.
func (a *Adapter) GetModelProfile(ctx context.Context, model string) (service.ModelProfile, error) {
	// Normalize model name
	model = strings.ToLower(model)

	profile, ok := a.profiles[model]
	if !ok {
		return service.ModelProfile{}, fmt.Errorf("unknown model: %s", model)
	}

	return profile, nil
}

// registerBuiltInProfiles registers built-in model profiles.
func (a *Adapter) registerBuiltInProfiles() {
	// Claude models
	a.profiles["claude-3-5-sonnet"] = service.ModelProfile{
		ContextWindow:    200000,
		InstructionStyle:  "xml_preference",
		FewShotCapacity:   5,
		TemperatureCurve:  "linear",
		SystemRoleSupport: true,
		JSONReliability:   0.95,
	}

	a.profiles["claude-3-opus"] = service.ModelProfile{
		ContextWindow:    200000,
		InstructionStyle:  "xml_preference",
		FewShotCapacity:   5,
		TemperatureCurve:  "linear",
		SystemRoleSupport: true,
		JSONReliability:   0.95,
	}

	a.profiles["claude-sonnet-4-20250514"] = service.ModelProfile{
		ContextWindow:    200000,
		InstructionStyle:  "xml_preference",
		FewShotCapacity:   5,
		TemperatureCurve:  "linear",
		SystemRoleSupport: true,
		JSONReliability:   0.95,
	}

	// GPT models
	a.profiles["gpt-4o"] = service.ModelProfile{
		ContextWindow:    128000,
		InstructionStyle:  "markdown_preference",
		FewShotCapacity:   3,
		TemperatureCurve:  "steep",
		SystemRoleSupport: true,
		JSONReliability:   0.9,
	}

	a.profiles["gpt-4o-mini"] = service.ModelProfile{
		ContextWindow:    128000,
		InstructionStyle:  "markdown_preference",
		FewShotCapacity:   3,
		TemperatureCurve:  "steep",
		SystemRoleSupport: true,
		JSONReliability:   0.85,
	}

	a.profiles["gpt-4-turbo"] = service.ModelProfile{
		ContextWindow:    128000,
		InstructionStyle:  "markdown_preference",
		FewShotCapacity:   3,
		TemperatureCurve:  "steep",
		SystemRoleSupport: true,
		JSONReliability:   0.9,
	}

	// Ollama models (generic)
	a.profiles["llama3"] = service.ModelProfile{
		ContextWindow:    8192,
		InstructionStyle:  "explicit_preference",
		FewShotCapacity:   2,
		TemperatureCurve:  "flat",
		SystemRoleSupport: true,
		JSONReliability:   0.7,
	}

	a.profiles["mistral"] = service.ModelProfile{
		ContextWindow:    8192,
		InstructionStyle:  "explicit_preference",
		FewShotCapacity:   2,
		TemperatureCurve:  "flat",
		SystemRoleSupport: true,
		JSONReliability:   0.7,
	}
}

// getDefaultProfile returns a default profile for unknown models.
func (a *Adapter) getDefaultProfile(model string) service.ModelProfile {
	return service.ModelProfile{
		ContextWindow:    4096,
		InstructionStyle:  "explicit_preference",
		FewShotCapacity:   2,
		TemperatureCurve:  "linear",
		SystemRoleSupport: true,
		JSONReliability:   0.75,
	}
}

// convertFormat converts prompt format from source to target model.
func (a *Adapter) convertFormat(prompt service.PromptContent, source, target service.ModelProfile) string {
	content := prompt.Instruction

	// Convert XML to Markdown if target prefers markdown
	if source.InstructionStyle == "xml_preference" && target.InstructionStyle == "markdown_preference" {
		content = a.xmlToMarkdown(content)
	}

	// Convert Markdown to XML if target prefers XML
	if source.InstructionStyle == "markdown_preference" && target.InstructionStyle == "xml_preference" {
		content = a.markdownToXML(content)
	}

	// Convert Examples format
	if len(prompt.Examples) > 0 {
		content = a.convertExamples(prompt.Examples, target.InstructionStyle)
	}

	return content
}

// xmlToMarkdown converts XML-formatted prompts to Markdown format.
func (a *Adapter) xmlToMarkdown(content string) string {
	// Convert <description> tags
	content = strings.ReplaceAll(content, "<description>", "**Description:**\n")
	content = strings.ReplaceAll(content, "</description>", "\n\n")

	// Convert <instruction> tags
	content = strings.ReplaceAll(content, "<instruction>", "**Instruction:**\n")
	content = strings.ReplaceAll(content, "</instruction>", "\n\n")

	// Convert <examples> tags
	content = strings.ReplaceAll(content, "<examples>", "**Examples:**\n")
	content = strings.ReplaceAll(content, "</examples>", "\n\n")

	// Convert <example> tags
	content = strings.ReplaceAll(content, "<example>", "- ")
	content = strings.ReplaceAll(content, "</example>", "\n")

	// Convert <input> tags
	content = strings.ReplaceAll(content, "<input>", "Input: ")
	content = strings.ReplaceAll(content, "</input>", "\n")

	// Convert <output> tags
	content = strings.ReplaceAll(content, "<output>", "Output: ")
	content = strings.ReplaceAll(content, "</output>", "\n")

	return content
}

// markdownToXML converts Markdown-formatted prompts to XML format.
func (a *Adapter) markdownToXML(content string) string {
	// Simple conversion for common patterns
	// Description
	content = strings.ReplaceAll(content, "**Description:**", "<description>")
	content = strings.ReplaceAll(content, "**Description:**\n", "<description>")

	// Instruction
	content = strings.ReplaceAll(content, "**Instruction:**", "<instruction>")
	content = strings.ReplaceAll(content, "**Instruction:**\n", "<instruction>")

	// Examples
	content = strings.ReplaceAll(content, "**Examples:**", "<examples>")
	content = strings.ReplaceAll(content, "**Examples:**\n", "<examples>")

	return content
}

// convertExamples converts examples to the target format.
func (a *Adapter) convertExamples(examples []service.Example, style string) string {
	var result strings.Builder
	result.WriteString("\n**Examples:**\n")

	for _, ex := range examples {
		switch style {
		case "xml_preference":
			result.WriteString("<example>\n")
			result.WriteString("<input>" + ex.Input + "</input>\n")
			result.WriteString("<output>" + ex.Output + "</output>\n")
			if ex.Footnote != "" {
				result.WriteString("<note>" + ex.Footnote + "</note>\n")
			}
			result.WriteString("</example>\n")
		case "markdown_preference":
			result.WriteString("- **Input:** " + ex.Input + "\n")
			result.WriteString("  **Output:** " + ex.Output + "\n")
			if ex.Footnote != "" {
				result.WriteString("  _" + ex.Footnote + "_\n")
			}
		default:
			result.WriteString("Input: " + ex.Input + "\nOutput: " + ex.Output + "\n")
		}
	}

	return result.String()
}

// applyParamAdjustments applies parameter adjustments for the target model.
func (a *Adapter) applyParamAdjustments(source, target service.ModelProfile, adjustments map[string]float64) {
	// Adjust few-shot capacity
	if target.FewShotCapacity < source.FewShotCapacity {
		adjustments["few_shot_delta"] = float64(target.FewShotCapacity - source.FewShotCapacity)
	}

	// Adjust temperature
	switch target.TemperatureCurve {
	case "steep":
		adjustments["temperature_multiplier"] = 0.9
	case "flat":
		adjustments["temperature_multiplier"] = 1.1
	}

	// Adjust max tokens if context window is smaller
	if target.ContextWindow < source.ContextWindow {
		adjustments["max_tokens_multiplier"] = float64(target.ContextWindow) / float64(source.ContextWindow)
	}
}

// checkContextLength checks if prompt fits in target model's context window.
func (a *Adapter) checkContextLength(prompt service.PromptContent, target service.ModelProfile, warnings *[]string) {
	// Estimate tokens (rough estimate: 4 chars per token)
	estimatedTokens := len(prompt.Instruction) / 4
	if len(prompt.Examples) > 0 {
		for _, ex := range prompt.Examples {
			estimatedTokens += len(ex.Input)/4 + len(ex.Output)/4
		}
	}

	// Reserve 20% for response
	availableTokens := int(float64(target.ContextWindow) * 0.8)

	if estimatedTokens > availableTokens {
		*warnings = append(*warnings, fmt.Sprintf("Prompt may be too long for target model context (est. %d tokens, available %d)", estimatedTokens, availableTokens))
	}
}

// NoopAdapter is a no-operation ModelAdapter.
type NoopAdapter struct{}

var errNoop = errors.New("modeladapter: noop adapter, enable the plugin")

// Adapt implements ModelAdapter.
func (n *NoopAdapter) Adapt(_ context.Context, _ service.PromptContent, _, _ string) (service.AdaptedPrompt, error) {
	return service.AdaptedPrompt{}, errNoop
}

// RecommendParams implements ModelAdapter.
func (n *NoopAdapter) RecommendParams(_ context.Context, _, _ string) (service.ModelParams, error) {
	return service.ModelParams{}, errNoop
}

// EstimateScore implements ModelAdapter.
func (n *NoopAdapter) EstimateScore(_ context.Context, _, _ string) (float64, error) {
	return 0, errNoop
}

// GetModelProfile implements ModelAdapter.
func (n *NoopAdapter) GetModelProfile(_ context.Context, _ string) (service.ModelProfile, error) {
	return service.ModelProfile{}, errNoop
}

// Ensure NoopAdapter implements service.ModelAdapter.
var _ service.ModelAdapter = (*NoopAdapter)(nil)
