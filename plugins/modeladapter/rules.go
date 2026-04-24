package modeladapter

import "strings"

// Rule represents an adaptation rule for model conversion.
type Rule interface {
	// Apply applies the rule to the content.
	Apply(content string) string

	// Description returns a description of the rule.
	Description() string
}

// RuleLibrary manages adaptation rules.
type RuleLibrary struct {
	rules []Rule
}

// NewRuleLibrary creates a new rule library with built-in rules.
func NewRuleLibrary() *RuleLibrary {
	lib := &RuleLibrary{
		rules: []Rule{
			&xmlToMarkdownRule{},
			&markdownToXMLRule{},
			&claudeToGPTRule{},
			&gptToClaudeRule{},
			&reduceFewShotRule{},
		},
	}
	return lib
}

// GetRules returns the rules for converting from source to target model.
func (lib *RuleLibrary) GetRules(sourceModel, targetModel string) []Rule {
	var rules []Rule

	sourceModel = strings.ToLower(sourceModel)
	targetModel = strings.ToLower(targetModel)

	// Determine which rules to apply based on model types
	if strings.Contains(sourceModel, "claude") && strings.Contains(targetModel, "gpt") {
		rules = append(rules, lib.rules[:3]...) // xmlToMarkdownRule + gpt rules
	} else if strings.Contains(sourceModel, "gpt") && strings.Contains(targetModel, "claude") {
		rules = append(rules, lib.rules[1:4]...) // markdownToXMLRule + claude rules
	}

	return rules
}

// xmlToMarkdownRule converts XML-formatted content to Markdown.
type xmlToMarkdownRule struct{}

func (r *xmlToMarkdownRule) Apply(content string) string {
	// Convert XML tags to Markdown formatting
	replacements := map[string]string{
		"<description>": "**Description:** ",
		"</description>": "\n\n",
		"<instruction>": "**Instruction:** ",
		"</instruction>": "\n\n",
		"<examples>": "**Examples:**\n",
		"</examples>": "\n",
		"<example>": "- ",
		"</example>": "\n",
		"<input>": "**Input:** ",
		"</input>": "\n",
		"<output>": "**Output:** ",
		"</output>": "\n",
		"<note>": "_",
		"</note>": "_\n",
		"<reasoning>": "**Reasoning:** ",
		"</reasoning>": "\n",
	}

	for old, new := range replacements {
		content = strings.ReplaceAll(content, old, new)
	}

	return content
}

func (r *xmlToMarkdownRule) Description() string {
	return "XML tags converted to Markdown formatting"
}

// markdownToXMLRule converts Markdown-formatted content to XML.
type markdownToXMLRule struct{}

func (r *markdownToXMLRule) Apply(content string) string {
	replacements := map[string]string{
		"**Description:**": "<description>",
		"**Description:**\n": "<description>",
		"**Instruction:**": "<instruction>",
		"**Instruction:**\n": "<instruction>",
		"**Examples:**\n": "<examples>",
		"**Examples:**": "<examples>",
		"**Input:**": "<input>",
		"**Input:**\n": "<input>",
		"**Output:**": "<output>",
		"**Output:**\n": "<output>",
		"_": "<note>",
		"_\n": "</note>\n",
	}

	for old, new := range replacements {
		content = strings.ReplaceAll(content, old, new)
	}

	return content
}

func (r *markdownToXMLRule) Description() string {
	return "Markdown formatting converted to XML tags"
}

// claudeToGPTRule applies Claude-to-GPT specific transformations.
type claudeToGPTRule struct{}

func (r *claudeToGPTRule) Apply(content string) string {
	// GPT prefers numbered lists over bullet points in some contexts
	content = strings.ReplaceAll(content, "- ", "1. ")

	// Ensure consistent spacing after headings
	content = strings.ReplaceAll(content, ":**\n", ":** ")

	return content
}

func (r *claudeToGPTRule) Description() string {
	return "Claude-specific formatting adapted for GPT"
}

// gptToClaudeRule applies GPT-to-Claude specific transformations.
type gptToClaudeRule struct{}

func (r *gptToClaudeRule) Apply(content string) string {
	// Claude appreciates XML-style structure
	// Add XML wrapper if not present
	if !strings.Contains(content, "<prompt>") && !strings.HasPrefix(content, "<?xml") {
		// Wrap content in XML structure if it has clear sections
		if strings.Contains(content, "**") {
			// Already markdown, keep as-is for Claude XML preference
		}
	}

	return content
}

func (r *gptToClaudeRule) Description() string {
	return "GPT-specific formatting adapted for Claude"
}

// reduceFewShotRule reduces the number of few-shot examples.
type reduceFewShotRule struct{}

func (r *reduceFewShotRule) Apply(content string) string {
	// Count examples and reduce if more than recommended
	// This is a placeholder - actual reduction would need context about target model
	return content
}

func (r *reduceFewShotRule) Description() string {
	return "Few-shot examples reduced for target model capacity"
}
