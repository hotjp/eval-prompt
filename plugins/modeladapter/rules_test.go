package modeladapter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuleLibrary_NewRuleLibrary(t *testing.T) {
	lib := NewRuleLibrary()
	require.NotNil(t, lib)
	require.Len(t, lib.rules, 5)
}

func TestRuleLibrary_GetRules(t *testing.T) {
	tests := []struct {
		name         string
		sourceModel  string
		targetModel  string
		wantRules    int
	}{
		{
			name:        "claude to gpt gets 3 rules",
			sourceModel: "claude-3-5-sonnet",
			targetModel: "gpt-4o",
			wantRules:   3,
		},
		{
			name:        "gpt to claude gets 3 rules",
			sourceModel: "gpt-4o",
			targetModel: "claude-3-5-sonnet",
			wantRules:   3,
		},
		{
			name:        "gpt to gpt gets no rules",
			sourceModel: "gpt-4o",
			targetModel: "gpt-4o",
			wantRules:   0,
		},
		{
			name:        "claude to claude gets no rules",
			sourceModel: "claude-sonnet-4-20250514",
			targetModel: "claude-3-opus",
			wantRules:   0,
		},
		{
			name:        "case insensitive source",
			sourceModel: "CLAUDE-3-5-SONNET",
			targetModel: "gpt-4o",
			wantRules:   3,
		},
		{
			name:        "case insensitive target",
			sourceModel: "claude-3-5-sonnet",
			targetModel: "GPT-4O",
			wantRules:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := NewRuleLibrary()
			rules := lib.GetRules(tt.sourceModel, tt.targetModel)
			require.Len(t, rules, tt.wantRules)
		})
	}
}

func TestXMLToMarkdownRule_Apply(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "description tag",
			input:    "<description>Test</description>",
			contains: "**Description:**",
		},
		{
			name:     "instruction tag",
			input:    "<instruction>Do this</instruction>",
			contains: "**Instruction:**",
		},
		{
			name:     "reasoning tag",
			input:    "<reasoning>思考过程</reasoning>",
			contains: "**Reasoning:**",
		},
		{
			name:     "note tag",
			input:    "<note>footnote</note>",
			contains: "_footnote_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &xmlToMarkdownRule{}
			result := rule.Apply(tt.input)
			require.Contains(t, result, tt.contains)
		})
	}
}

func TestXMLToMarkdownRule_Description(t *testing.T) {
	rule := &xmlToMarkdownRule{}
	desc := rule.Description()
	require.Contains(t, desc, "Markdown")
}

func TestMarkdownToXMLRule_Apply(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "description bold",
			input:    "**Description:** Test",
			contains: "<description>",
		},
		{
			name:     "instruction bold",
			input:    "**Instruction:** Do this",
			contains: "<instruction>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &markdownToXMLRule{}
			result := rule.Apply(tt.input)
			require.Contains(t, result, tt.contains)
		})
	}
}

func TestMarkdownToXMLRule_Description(t *testing.T) {
	rule := &markdownToXMLRule{}
	desc := rule.Description()
	require.Contains(t, desc, "XML")
}

func TestClaudeToGPTRule_Apply(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts bullet to numbered list",
			input:    "- item 1\n- item 2",
			expected: "1. item 1\n1. item 2",
		},
		{
			name:     "fixes spacing after bold colon",
			input:    "**Test:**\nvalue",
			expected: "**Test:** value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &claudeToGPTRule{}
			result := rule.Apply(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestClaudeToGPTRule_Description(t *testing.T) {
	rule := &claudeToGPTRule{}
	desc := rule.Description()
	require.Contains(t, desc, "Claude")
	require.Contains(t, desc, "GPT")
}

func TestGPTToClaudeRule_Apply(t *testing.T) {
	rule := &gptToClaudeRule{}
	result := rule.Apply("Some **markdown** content")
	// Should return content unchanged (placeholder implementation)
	require.NotEmpty(t, result)
}

func TestGPTToClaudeRule_Description(t *testing.T) {
	rule := &gptToClaudeRule{}
	desc := rule.Description()
	require.Contains(t, desc, "GPT")
	require.Contains(t, desc, "Claude")
}

func TestReduceFewShotRule_Apply(t *testing.T) {
	rule := &reduceFewShotRule{}
	result := rule.Apply("test content")
	// Placeholder returns content unchanged
	require.Equal(t, "test content", result)
}

func TestReduceFewShotRule_Description(t *testing.T) {
	rule := &reduceFewShotRule{}
	desc := rule.Description()
	require.Contains(t, desc, "Few-shot")
}

func TestRuleInterface_AllRulesImplement(t *testing.T) {
	// Verify all rules implement the Rule interface
	rules := []Rule{
		&xmlToMarkdownRule{},
		&markdownToXMLRule{},
		&claudeToGPTRule{},
		&gptToClaudeRule{},
		&reduceFewShotRule{},
	}

	for _, rule := range rules {
		// Apply should not panic
		result := rule.Apply("test input")
		require.NotEmpty(t, result)

		// Description should not panic
		desc := rule.Description()
		require.NotEmpty(t, desc)
	}
}
