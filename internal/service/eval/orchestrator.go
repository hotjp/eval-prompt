// Package eval provides the evaluation orchestrator and plugin system.
//
// Orchestrator coordinates the execution of evaluation plugins across multiple
// test cases, manages parallelism, calculates confidence intervals, performs
// baseline comparisons, and updates ELO ratings.
//
// The orchestration flow:
//  1. Load Asset + TestCases
//  2. Apply InjectionStrategy to generate variants
//  3. Execute all Plugins in parallel (errgroup with concurrency limit)
//  4. Collect results and calculate confidence intervals
//  5. Compare against baseline
//  6. Update ELO ratings
package eval

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/eval-prompt/internal/service/eval/stats"
)

// EvalConfig contains configuration for the evaluation orchestrator.
type EvalConfig struct {
	// Plugins is the list of plugin names to execute.
	Plugins []string
	// InjectionStrategy is the name of the injection strategy to use.
	InjectionStrategy string
	// StatsConfig contains statistical configuration.
	StatsConfig StatsConfig
	// Parallelism controls the maximum number of concurrent plugin executions.
	Parallelism int
}

// StatsConfig contains configuration for statistical calculations.
type StatsConfig struct {
	// ConfidenceLevel is the confidence level for CI calculation (e.g., 0.95).
	ConfidenceLevel float64
	// BootstrapIterations is the number of bootstrap iterations for CI.
	BootstrapIterations int
	// BaselineID is the ID of the baseline snapshot to compare against.
	BaselineID string
}

// Orchestrator coordinates evaluation plugin execution.
type Orchestrator struct {
	registry *Registry
	config   EvalConfig
	logger   *slog.Logger
}

// NewOrchestrator creates a new evaluation orchestrator.
func NewOrchestrator(registry *Registry, config EvalConfig, logger *slog.Logger) *Orchestrator {
	if config.Parallelism <= 0 {
		config.Parallelism = 4 // default parallelism
	}
	if config.StatsConfig.BootstrapIterations <= 0 {
		config.StatsConfig.BootstrapIterations = 1000
	}
	if config.StatsConfig.ConfidenceLevel <= 0 {
		config.StatsConfig.ConfidenceLevel = 0.95
	}
	return &Orchestrator{
		registry: registry,
		config:   config,
		logger:   logger,
	}
}

// OrchestratorResult contains the final result of orchestration.
type OrchestratorResult struct {
	// OverallScore is the aggregate score across all plugins.
	OverallScore float64
	// PluginResults maps plugin name to its result.
	PluginResults map[string]PluginExecutionResult
	// ConfidenceInterval is the 95% confidence interval for the overall score.
	ConfidenceInterval *ConfidenceInterval
	// BaselineComparison contains the comparison against baseline.
	BaselineComparison *BaselineComparison
	// ELOResult contains updated ELO ratings.
	ELOResult *ELOResult
	// Summary is a human-readable summary.
	Summary string
}

// PluginExecutionResult contains the result of a single plugin execution.
type PluginExecutionResult struct {
	// PluginName is the name of the plugin.
	PluginName string
	// Score is the aggregate score from this plugin.
	Score float64
	// ConfidenceInterval is the CI for this plugin's score.
	ConfidenceInterval *ConfidenceInterval
	// WorkItemResults contains results for each work item.
	WorkItemResults []WorkItemResult
	// Error contains any error that occurred.
	Error error
}

// WorkItemResult contains the result of evaluating a single work item.
type WorkItemResult struct {
	// WorkItemID is the work item identifier.
	WorkItemID string
	// Score is the score for this work item.
	Score float64
	// Details contains plugin-specific details.
	Details map[string]any
	// DurationMs is the execution duration in milliseconds.
	DurationMs int64
}

// BaselineComparison contains the comparison against a baseline.
type BaselineComparison struct {
	// BaselineID is the baseline snapshot ID.
	BaselineID string
	// ScoreDelta is the difference in scores (current - baseline).
	ScoreDelta float64
	// EffectSize is Cohen's d effect size.
	EffectSize float64
	// EffectInterpretation is a human-readable interpretation of effect size.
	EffectInterpretation string
	// TStat is the t-statistic from paired t-test.
	TStat float64
	// PValue is the p-value from paired t-test.
	PValue float64
	// IsSignificant indicates whether the difference is statistically significant.
	IsSignificant bool
}

// ELOResult contains the ELO rating update result.
type ELOResult struct {
	// NewRating is the updated ELO rating.
	NewRating float64
	// PreviousRating is the rating before update.
	PreviousRating float64
	// Outcome is the match outcome (1.0=win, 0.0=loss, 0.5=draw).
	Outcome float64
}

// defaultInjectionStrategy applies no transformation.
type defaultInjectionStrategy struct{}

// Apply implements InjectionStrategy by returning the input unchanged.
func (s *defaultInjectionStrategy) Apply(input EvalInput) []EvalInput {
	return []EvalInput{input}
}

// Name implements InjectionStrategy.
func (s *defaultInjectionStrategy) Name() string {
	return "default"
}

// Run executes the orchestration flow.
func (o *Orchestrator) Run(ctx context.Context, assetID, snapshotID string, testCases []*TestCase, baselineScores map[string][]float64) (*OrchestratorResult, error) {
	startTime := time.Now()
	o.logger.Info("starting orchestration", "asset_id", assetID, "snapshot_id", snapshotID, "test_cases", len(testCases), "plugins", len(o.config.Plugins))

	// Get injection strategy
	strategy := o.getInjectionStrategy()

	// Generate work items by applying injection strategy to each test case
	workItems := o.generateWorkItems(testCases, strategy)
	o.logger.Info("generated work items", "count", len(workItems))

	// Execute plugins in parallel
	pluginResults, err := o.executePlugins(ctx, workItems)
	if err != nil {
		return nil, fmt.Errorf("plugin execution failed: %w", err)
	}

	// Calculate confidence intervals
	ciResult := o.calculateConfidenceIntervals(pluginResults)

	// Compare against baseline if configured
	var baselineComparison *BaselineComparison
	if o.config.StatsConfig.BaselineID != "" && baselineScores != nil {
		baselineComparison = o.compareWithBaseline(pluginResults, baselineScores)
	}

	// Calculate overall score
	overallScore := o.calculateOverallScore(pluginResults)

	// Calculate ELO update
	eloResult := o.calculateELO(pluginResults, baselineScores, baselineComparison)

	result := &OrchestratorResult{
		OverallScore:       overallScore,
		PluginResults:      pluginResults,
		ConfidenceInterval: ciResult,
		BaselineComparison: baselineComparison,
		ELOResult:          eloResult,
		Summary:            o.generateSummary(overallScore, ciResult, baselineComparison),
	}

	elapsed := time.Since(startTime)
	o.logger.Info("orchestration completed", "duration_ms", elapsed.Milliseconds(), "overall_score", overallScore)

	return result, nil
}

// getInjectionStrategy returns the injection strategy based on configuration.
func (o *Orchestrator) getInjectionStrategy() InjectionStrategy {
	switch o.config.InjectionStrategy {
	case "default", "":
		return &defaultInjectionStrategy{}
	case "position_swap":
		return &PositionSwap{}
	case "constraint_conflict":
		return &ConstraintConflict{}
	case "adversarial_prefix":
		return &AdversarialPrefix{}
	default:
		o.logger.Warn("unknown injection strategy, using default", "strategy", o.config.InjectionStrategy)
		return &defaultInjectionStrategy{}
	}
}

// workItem represents a single evaluation work item.
type workItem struct {
	ID       string
	TestCase *TestCase
	Prompt   string
}

// generateWorkItems generates work items from test cases using the injection strategy.
func (o *Orchestrator) generateWorkItems(testCases []*TestCase, strategy InjectionStrategy) []workItem {
	var items []workItem
	for _, tc := range testCases {
		// Convert TestCase to EvalInput
		input := EvalInput{
			AssetID:   tc.ID,
			Candidate: tc.Prompt,
			Reference: tc.Expected,
			TestCase:  tc,
		}

		// Apply injection strategy
		variants := strategy.Apply(input)
		for i, variant := range variants {
			itemID := fmt.Sprintf("%s-%d", tc.ID, i)
			items = append(items, workItem{
				ID:       itemID,
				TestCase: tc,
				Prompt:   variant.Candidate,
			})
		}
	}
	return items
}

// executePlugins runs all configured plugins in parallel with concurrency limiting.
func (o *Orchestrator) executePlugins(ctx context.Context, workItems []workItem) (map[string]PluginExecutionResult, error) {
	results := make(map[string]PluginExecutionResult)
	var mu sync.Mutex
	var groupErr error

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(o.config.Parallelism)

	for _, pluginName := range o.config.Plugins {
		pluginName := pluginName // capture loop variable
		g.Go(func() error {
			result, err := o.executePlugin(ctx, pluginName, workItems)
			mu.Lock()
			results[pluginName] = result
			if err != nil && groupErr == nil {
				groupErr = err
			}
			mu.Unlock()
			return nil // We handle errors ourselves to continue other plugins
		})
	}

	if err := g.Wait(); err != nil {
		o.logger.Error("errgroup error", "error", err)
	}

	return results, groupErr
}

// executePlugin runs a single plugin with timeout on each work item.
func (o *Orchestrator) executePlugin(ctx context.Context, pluginName string, workItems []workItem) (PluginExecutionResult, error) {
	plugin, err := Get(pluginName)
	if err != nil {
		return PluginExecutionResult{
			PluginName: pluginName,
			Error:      err,
		}, err
	}

	result := PluginExecutionResult{
		PluginName:      pluginName,
		WorkItemResults: make([]WorkItemResult, 0, len(workItems)),
	}

	// Execute work items in parallel with limited concurrency
	workItemChan := make(chan workItem, len(workItems))
	resultChan := make(chan WorkItemResult, len(workItems))

	// Start worker pool
	var wg sync.WaitGroup
	workerCount := o.config.Parallelism
	if workerCount <= 0 {
		workerCount = 4
	}
	if workerCount > len(workItems) {
		workerCount = len(workItems)
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range workItemChan {
				itemResult, _ := o.executeWorkItem(ctx, plugin, item)
				resultChan <- itemResult
			}
		}()
	}

	// Feed work items to workers
	for _, item := range workItems {
		workItemChan <- item
	}
	close(workItemChan)

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)

	// Collect results
	for itemResult := range resultChan {
		result.WorkItemResults = append(result.WorkItemResults, itemResult)
	}

	// Calculate aggregate score
	result.Score = o.calculatePluginScore(result.WorkItemResults)

	// Calculate per-plugin confidence interval
	if len(result.WorkItemResults) > 0 {
		scores := make([]float64, len(result.WorkItemResults))
		for i, wr := range result.WorkItemResults {
			scores[i] = wr.Score
		}
		low, high := stats.BootstrapCI(scores, o.config.StatsConfig.ConfidenceLevel, o.config.StatsConfig.BootstrapIterations)
		result.ConfidenceInterval = &ConfidenceInterval{Low: low, High: high}
	}

	return result, nil
}

// executeWorkItem executes a single work item with timeout.
func (o *Orchestrator) executeWorkItem(ctx context.Context, plugin EvalPlugin, item workItem) (WorkItemResult, error) {
	// Default timeout per work item: 30 seconds
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	evalInput := EvalInput{
		AssetID:   item.TestCase.ID,
		Candidate: item.Prompt,
		Reference: item.TestCase.Expected,
		TestCase:  item.TestCase,
		Metadata: map[string]any{
			"work_item_id": item.ID,
		},
	}

	evalResult := plugin.Evaluate(ctx, evalInput)
	duration := time.Since(start)

	score := evalResult.Score
	if math.IsNaN(score) || math.IsInf(score, 0) {
		score = 0.0
	}

	return WorkItemResult{
		WorkItemID: item.ID,
		Score:      score,
		Details:    evalResult.Details,
		DurationMs: duration.Milliseconds(),
	}, nil
}

// calculatePluginScore calculates the aggregate score for a plugin's work items.
func (o *Orchestrator) calculatePluginScore(results []WorkItemResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	var sum, count float64
	for _, r := range results {
		sum += r.Score
		count++
	}

	if count == 0 {
		return 0.0
	}

	return sum / count
}

// calculateConfidenceIntervals calculates overall CI from per-plugin scores.
func (o *Orchestrator) calculateConfidenceIntervals(pluginResults map[string]PluginExecutionResult) *ConfidenceInterval {
	// Collect per-plugin aggregate scores
	var pluginScores []float64
	for _, pr := range pluginResults {
		if len(pr.WorkItemResults) > 0 {
			pluginScores = append(pluginScores, pr.Score)
		}
	}

	if len(pluginScores) == 0 {
		return &ConfidenceInterval{Low: 0, High: 0}
	}

	low, high := stats.BootstrapCI(pluginScores, o.config.StatsConfig.ConfidenceLevel, o.config.StatsConfig.BootstrapIterations)
	return &ConfidenceInterval{Low: low, High: high}
}

// compareWithBaseline compares current results against baseline scores.
func (o *Orchestrator) compareWithBaseline(pluginResults map[string]PluginExecutionResult, baselineScores map[string][]float64) *BaselineComparison {
	var currentScores []float64
	for _, pr := range pluginResults {
		for _, wr := range pr.WorkItemResults {
			currentScores = append(currentScores, wr.Score)
		}
	}

	// Aggregate baseline scores
	var baseline []float64
	for _, scores := range baselineScores {
		baseline = append(baseline, scores...)
	}

	if len(currentScores) == 0 || len(baseline) == 0 {
		return nil
	}

	// Ensure same length for paired t-test (sample if needed)
	if len(currentScores) != len(baseline) {
		// Use minimum length
		minLen := len(currentScores)
		if minLen > len(baseline) {
			minLen = len(baseline)
		}
		currentScores = currentScores[:minLen]
		baseline = baseline[:minLen]
	}

	// Calculate effect size
	effectSize := stats.CohensD(currentScores, baseline)

	// Perform paired t-test
	tStat, pValue := stats.PairedTTest(baseline, currentScores)

	// Calculate score delta
	var currentMean, baselineMean float64
	for _, s := range currentScores {
		currentMean += s
	}
	currentMean /= float64(len(currentScores))
	for _, s := range baseline {
		baselineMean += s
	}
	baselineMean /= float64(len(baseline))

	significanceThreshold := 1.0 - o.config.StatsConfig.ConfidenceLevel
	if math.IsNaN(pValue) {
		significanceThreshold = 0.05
	}

	return &BaselineComparison{
		BaselineID:           o.config.StatsConfig.BaselineID,
		ScoreDelta:           currentMean - baselineMean,
		EffectSize:           effectSize,
		EffectInterpretation: stats.InterpretCohensD(effectSize),
		TStat:                tStat,
		PValue:               pValue,
		IsSignificant:        pValue < significanceThreshold,
	}
}

// calculateOverallScore calculates the overall score across all plugins.
func (o *Orchestrator) calculateOverallScore(pluginResults map[string]PluginExecutionResult) float64 {
	if len(pluginResults) == 0 {
		return 0.0
	}

	var sum, count float64
	for _, pr := range pluginResults {
		sum += pr.Score
		count++
	}

	if count == 0 {
		return 0.0
	}

	return sum / count
}

// calculateELO calculates ELO rating update based on comparison with baseline.
func (o *Orchestrator) calculateELO(pluginResults map[string]PluginExecutionResult, baselineScores map[string][]float64, baselineComparison *BaselineComparison) *ELOResult {
	// If no baseline, use a default rating of 1500
	const defaultRating = 1500.0
	currentRating := defaultRating

	if baselineScores != nil {
		// Calculate average baseline score to determine outcome
		var baselineMean float64
		var count int
		for _, scores := range baselineScores {
			for _, s := range scores {
				baselineMean += s
				count++
			}
		}
		if count > 0 {
			baselineMean /= float64(count)
			// Use a fixed baseline rating for comparison
			currentRating = baselineMean * 1000 // Scale to ELO-like rating
		}
	}

	// Calculate current performance
	var currentMean float64
	var itemCount int
	for _, pr := range pluginResults {
		for _, wr := range pr.WorkItemResults {
			currentMean += wr.Score
			itemCount++
		}
	}
	if itemCount > 0 {
		currentMean /= float64(itemCount)
	}

	// Determine outcome based on current vs baseline
	var outcome float64
	if baselineComparison != nil {
		// Use actual baseline comparison to determine outcome
		// ScoreDelta > 0 means current is better than baseline
		if baselineComparison.ScoreDelta > 0.05 { // meaningful improvement
			outcome = 1.0 // win
		} else if baselineComparison.ScoreDelta < -0.05 { // meaningful degradation
			outcome = 0.0 // loss
		} else {
			outcome = 0.5 // draw/neutral
		}
	} else {
		outcome = 0.5 // neutral when no baseline
	}

	newRating, _ := stats.UpdateELO(currentRating, defaultRating, outcome)

	return &ELOResult{
		NewRating:      newRating,
		PreviousRating: currentRating,
		Outcome:        outcome,
	}
}

// generateSummary creates a human-readable summary.
func (o *Orchestrator) generateSummary(overallScore float64, ci *ConfidenceInterval, baseline *BaselineComparison) string {
	summary := fmt.Sprintf("Overall Score: %.4f (95%% CI: [%.4f, %.4f])", overallScore, ci.Low, ci.High)

	if baseline != nil {
		summary += fmt.Sprintf("\nBaseline Comparison (ID: %s):", baseline.BaselineID)
		summary += fmt.Sprintf("\n  Score Delta: %.4f", baseline.ScoreDelta)
		summary += fmt.Sprintf("\n  Effect Size: %.4f (%s)", baseline.EffectSize, baseline.EffectInterpretation)
		if baseline.IsSignificant {
			summary += "\n  Statistically Significant: Yes"
		} else {
			summary += "\n  Statistically Significant: No"
		}
	}

	return summary
}
