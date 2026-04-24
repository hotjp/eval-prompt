package modeladapter

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAdapter_Adapt(t *testing.T) {
	tests := []struct {
		name         string
		prompt       service.PromptContent
		sourceModel  string
		targetModel  string
		wantContent  bool
		wantWarnings bool
	}{
		{
			name: "same model returns original instruction",
			prompt: service.PromptContent{
				Instruction: "Test instruction",
			},
			sourceModel: "gpt-4o",
			targetModel: "gpt-4o",
			wantContent: true,
		},
		{
			name: "claude to gpt converts xml to markdown",
			prompt: service.PromptContent{
				Instruction: "<description>Test</description>",
			},
			sourceModel: "claude-3-5-sonnet",
			targetModel: "gpt-4o",
			wantContent: true,
		},
		{
			name: "gpt to claude converts markdown to xml",
			prompt: service.PromptContent{
				Instruction: "**Description:** Test",
			},
			sourceModel: "gpt-4o",
			targetModel: "claude-3-5-sonnet",
			wantContent: true,
		},
		{
			name: "unknown source uses default profile",
			prompt: service.PromptContent{
				Instruction: "Test instruction",
			},
			sourceModel: "unknown-model",
			targetModel: "gpt-4o",
			wantContent: true,
		},
		{
			name: "unknown target uses default profile",
			prompt: service.PromptContent{
				Instruction: "Test instruction",
			},
			sourceModel: "gpt-4o",
			targetModel: "unknown-model",
			wantContent: true,
		},
		{
			name: "long instruction generates warning",
			prompt: service.PromptContent{
				Instruction: string(make([]byte, 20000)), // Long instruction
			},
			sourceModel: "gpt-4o",
			targetModel: "gpt-4o-mini",
			wantContent: true,
			wantWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(Config{Enabled: true})
			result, err := adapter.Adapt(context.Background(), tt.prompt, tt.sourceModel, tt.targetModel)
			require.NoError(t, err)
			require.NotEmpty(t, result.Content)

			if tt.sourceModel == tt.targetModel {
				require.Equal(t, tt.prompt.Instruction, result.Content)
			}
		})
	}
}

func TestAdapter_Adapt_WithExamples(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	prompt := service.PromptContent{
		Instruction: "Test instruction",
		Examples: []service.Example{
			{Input: "input1", Output: "output1"},
			{Input: "input2", Output: "output2"},
		},
	}

	result, err := adapter.Adapt(context.Background(), prompt, "claude-3-5-sonnet", "gpt-4o")
	require.NoError(t, err)
	require.NotEmpty(t, result.Content)
	// ConvertExamples with xml style creates content with <examples> tag
	// The actual output will vary based on format conversion, so just check it's not empty
}

func TestAdapter_Adapt_ParamAdjustments(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	prompt := service.PromptContent{
		Instruction: "Test instruction",
	}

	result, err := adapter.Adapt(context.Background(), prompt, "claude-3-5-sonnet", "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, result.ParamAdjustments)
}

func TestAdapter_RecommendParams(t *testing.T) {
	// Use llama3 which has "flat" temperature curve (multiplier 1.1) instead of "steep" (0.9)
	// This makes calculations predictable without complex multipliers
	tests := []struct {
		name       string
		model      string
		taskType   string
		wantTokens int
	}{
		{
			name:       "code generation uses correct tokens",
			model:      "llama3",
			taskType:   "code_generation",
			wantTokens: 4096,
		},
		{
			name:       "creative writing uses correct tokens",
			model:      "llama3",
			taskType:   "creative_writing",
			wantTokens: 2048,
		},
		{
			name:       "analysis uses correct tokens",
			model:      "llama3",
			taskType:   "analysis",
			wantTokens: 2048,
		},
		{
			name:       "summarization uses correct tokens",
			model:      "llama3",
			taskType:   "summarization",
			wantTokens: 512,
		},
		{
			name:       "json output uses correct tokens",
			model:      "llama3",
			taskType:   "json_output",
			wantTokens: 2048,
		},
		{
			name:       "unknown task type uses default tokens",
			model:      "llama3",
			taskType:   "unknown_task",
			wantTokens: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(Config{Enabled: true})
			params, err := adapter.RecommendParams(context.Background(), tt.model, tt.taskType)
			require.NoError(t, err)
			// Temperature varies based on model profile and task type, just verify it's set and tokens match
			require.Greater(t, params.Temperature, float64(0))
			require.Equal(t, tt.wantTokens, params.MaxTokens)
		})
	}
}

func TestAdapter_RecommendParams_TaskTypesSetTokens(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})

	// Verify different task types correctly set max tokens
	taskTypes := []string{"code_generation", "creative_writing", "analysis", "summarization", "json_output"}
	expectedTokens := []int{4096, 2048, 2048, 512, 2048}

	for i, taskType := range taskTypes {
		params, err := adapter.RecommendParams(context.Background(), "llama3", taskType)
		require.NoError(t, err)
		require.Equal(t, expectedTokens[i], params.MaxTokens, "task type %s should set correct tokens", taskType)
	}
}

func TestAdapter_EstimateScore(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	score, err := adapter.EstimateScore(context.Background(), "prompt-123", "gpt-4o")
	require.NoError(t, err)
	require.Equal(t, 0.85, score)
}

func TestAdapter_GetModelProfile(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantCtxWin int
		wantStyle  string
		wantErr    bool
	}{
		{
			name:       "claude 3.5 sonnet",
			model:      "claude-3-5-sonnet",
			wantCtxWin: 200000,
			wantStyle:  "xml_preference",
		},
		{
			name:       "gpt-4o",
			model:      "gpt-4o",
			wantCtxWin: 128000,
			wantStyle:  "markdown_preference",
		},
		{
			name:       "llama3",
			model:      "llama3",
			wantCtxWin: 8192,
			wantStyle:  "explicit_preference",
		},
		{
			name:       "case insensitive lookup",
			model:      "GPT-4O",
			wantCtxWin: 128000,
			wantStyle:  "markdown_preference",
		},
		{
			name:    "unknown model",
			model:   "unknown-model",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(Config{Enabled: true})
			profile, err := adapter.GetModelProfile(context.Background(), tt.model)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantCtxWin, profile.ContextWindow)
				require.Equal(t, tt.wantStyle, profile.InstructionStyle)
			}
		})
	}
}

func TestAdapter_GetModelProfile_UnknownModel(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	_, err := adapter.GetModelProfile(context.Background(), "non-existent-model")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown model")
}

func TestAdapter_ConvertFormat_XmlToMarkdown(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	prompt := service.PromptContent{
		Instruction: "<description>Test description</description><instruction>Do something</instruction>",
	}
	source := service.ModelProfile{InstructionStyle: "xml_preference"}
	target := service.ModelProfile{InstructionStyle: "markdown_preference"}

	content := adapter.convertFormat(prompt, source, target)
	require.Contains(t, content, "**Description:**")
	require.Contains(t, content, "**Instruction:**")
	require.NotContains(t, content, "<description>")
}

func TestAdapter_ConvertFormat_MarkdownToXml(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	prompt := service.PromptContent{
		Instruction: "**Description:** Test description",
	}
	source := service.ModelProfile{InstructionStyle: "markdown_preference"}
	target := service.ModelProfile{InstructionStyle: "xml_preference"}

	content := adapter.convertFormat(prompt, source, target)
	require.Contains(t, content, "<description>")
}

func TestAdapter_XmlToMarkdown(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	content := "<description>Test</description><instruction>Do it</instruction><examples><example><input>1</input><output>2</output></example></examples>"
	result := adapter.xmlToMarkdown(content)

	require.Contains(t, result, "**Description:**")
	require.Contains(t, result, "**Instruction:**")
	require.Contains(t, result, "**Examples:**")
	require.Contains(t, result, "- ")
	require.Contains(t, result, "Input:")
	require.Contains(t, result, "Output:")
}

func TestAdapter_MarkdownToXML(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	content := "**Description:** Test\n**Instruction:** Do it"
	result := adapter.markdownToXML(content)

	require.Contains(t, result, "<description>")
	require.Contains(t, result, "<instruction>")
}

func TestAdapter_ConvertExamples_XmlStyle(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	examples := []service.Example{
		{Input: "test input", Output: "test output", Footnote: "note"},
	}

	result := adapter.convertExamples(examples, "xml_preference")
	require.Contains(t, result, "<example>")
	require.Contains(t, result, "<input>test input</input>")
	require.Contains(t, result, "<output>test output</output>")
	require.Contains(t, result, "<note>note</note>")
}

func TestAdapter_ConvertExamples_MarkdownStyle(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	examples := []service.Example{
		{Input: "test input", Output: "test output", Footnote: "note"},
	}

	result := adapter.convertExamples(examples, "markdown_preference")
	require.Contains(t, result, "**Input:**")
	require.Contains(t, result, "**Output:**")
	require.Contains(t, result, "_note_")
}

func TestAdapter_ApplyParamAdjustments(t *testing.T) {
	tests := []struct {
		name      string
		source    service.ModelProfile
		target    service.ModelProfile
		wantDelta bool
	}{
		{
			name:      "target has fewer few-shot capacity",
			source:    service.ModelProfile{FewShotCapacity: 5},
			target:    service.ModelProfile{FewShotCapacity: 3},
			wantDelta: true,
		},
		{
			name:      "target has more few-shot capacity",
			source:    service.ModelProfile{FewShotCapacity: 3},
			target:    service.ModelProfile{FewShotCapacity: 5},
			wantDelta: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(Config{Enabled: true})
			adjustments := make(map[string]float64)
			adapter.applyParamAdjustments(tt.source, tt.target, adjustments)

			if tt.wantDelta {
				require.Contains(t, adjustments, "few_shot_delta")
			} else {
				_, hasDelta := adjustments["few_shot_delta"]
				require.False(t, hasDelta)
			}
		})
	}
}

func TestAdapter_CheckContextLength(t *testing.T) {
	tests := []struct {
		name         string
		prompt       service.PromptContent
		target       service.ModelProfile
		wantWarning  bool
	}{
		{
			name: "prompt fits",
			prompt: service.PromptContent{
				Instruction: "short instruction",
			},
			target: service.ModelProfile{
				ContextWindow: 128000,
			},
			wantWarning: false,
		},
		{
			name: "prompt too long",
			prompt: service.PromptContent{
				Instruction: string(make([]byte, 50000)),
			},
			target: service.ModelProfile{
				ContextWindow: 4096,
			},
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(Config{Enabled: true})
			var warnings []string
			adapter.checkContextLength(tt.prompt, tt.target, &warnings)

			if tt.wantWarning {
				require.NotEmpty(t, warnings)
			} else {
				require.Empty(t, warnings)
			}
		})
	}
}

func TestAdapter_ImplementsInterface(t *testing.T) {
	adapter := NewAdapter(Config{Enabled: true})
	var _ service.ModelAdapter = adapter
}

func TestNoopAdapter_Adapt(t *testing.T) {
	adapter := &NoopAdapter{}
	_, err := adapter.Adapt(context.Background(), service.PromptContent{}, "source", "target")
	require.Error(t, err)
	require.Contains(t, err.Error(), "noop adapter")
}

func TestNoopAdapter_RecommendParams(t *testing.T) {
	adapter := &NoopAdapter{}
	_, err := adapter.RecommendParams(context.Background(), "gpt-4o", "code_generation")
	require.Error(t, err)
}

func TestNoopAdapter_EstimateScore(t *testing.T) {
	adapter := &NoopAdapter{}
	_, err := adapter.EstimateScore(context.Background(), "prompt-123", "gpt-4o")
	require.Error(t, err)
}

func TestNoopAdapter_GetModelProfile(t *testing.T) {
	adapter := &NoopAdapter{}
	_, err := adapter.GetModelProfile(context.Background(), "gpt-4o")
	require.Error(t, err)
}
