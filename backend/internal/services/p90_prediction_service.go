package services

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"
	
	"claudeee-backend/internal/models"
)

type P90PredictionService struct {
	db *sql.DB
}

type P90Prediction struct {
	TokenLimit    float64 `json:"token_limit"`
	MessageLimit  int     `json:"message_limit"`
	CostLimit     float64 `json:"cost_limit"`
	Confidence    float64 `json:"confidence"`
	TimeToLimit   int     `json:"time_to_limit_minutes"`
	BurnRate      float64 `json:"burn_rate_per_hour"`
	PredictedAt   time.Time `json:"predicted_at"`
}

type UsageMetrics struct {
	Tokens    []float64
	Messages  []int
	Costs     []float64
	Timestamp time.Time
}

func NewP90PredictionService(db *sql.DB) *P90PredictionService {
	return &P90PredictionService{
		db: db,
	}
}

const (
	PREDICTION_WINDOW_HOURS = 192 // 8 days for historical analysis
	CONFIDENCE_FACTOR       = 0.95
	MIN_DATA_POINTS        = 10
)

// CalculateP90Limits calculates p90 limits for token, message, and cost usage
func (s *P90PredictionService) CalculateP90Limits() (*P90Prediction, error) {
	// Get historical data for the last 8 days
	metrics, err := s.getHistoricalMetrics(PREDICTION_WINDOW_HOURS)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical metrics: %w", err)
	}

	// Check if we have enough data points
	if len(metrics.Tokens) < MIN_DATA_POINTS {
		return nil, fmt.Errorf("insufficient data points for prediction: %d (need at least %d)", len(metrics.Tokens), MIN_DATA_POINTS)
	}

	// Calculate p90 values for each metric
	tokenP90 := s.calculatePercentile(metrics.Tokens, 90)
	messageP90 := s.calculatePercentileInt(metrics.Messages, 90)
	costP90 := s.calculatePercentile(metrics.Costs, 90)

	// Apply confidence factor
	prediction := &P90Prediction{
		TokenLimit:   tokenP90 * CONFIDENCE_FACTOR,
		MessageLimit: int(float64(messageP90) * CONFIDENCE_FACTOR),
		CostLimit:    costP90 * CONFIDENCE_FACTOR,
		Confidence:   CONFIDENCE_FACTOR,
		PredictedAt:  time.Now().UTC(),
	}

	// Calculate burn rate from recent data (last 1 hour)
	burnRate, err := s.calculateBurnRate()
	if err != nil {
		// If burn rate calculation fails, set to 0 but don't fail the entire prediction
		burnRate = 0
	}
	prediction.BurnRate = burnRate

	// Calculate time to limit based on current usage and burn rate
	timeToLimit, err := s.calculateTimeToLimit(prediction.TokenLimit, burnRate)
	if err != nil {
		timeToLimit = 0 // Set to 0 if calculation fails
	}
	prediction.TimeToLimit = timeToLimit

	return prediction, nil
}

// GetP90LimitsByProject calculates p90 limits for a specific project
func (s *P90PredictionService) GetP90LimitsByProject(projectName string) (*P90Prediction, error) {
	metrics, err := s.getHistoricalMetricsByProject(projectName, PREDICTION_WINDOW_HOURS)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical metrics for project %s: %w", projectName, err)
	}

	if len(metrics.Tokens) < MIN_DATA_POINTS {
		return nil, fmt.Errorf("insufficient data points for project %s: %d (need at least %d)", projectName, len(metrics.Tokens), MIN_DATA_POINTS)
	}

	tokenP90 := s.calculatePercentile(metrics.Tokens, 90)
	messageP90 := s.calculatePercentileInt(metrics.Messages, 90)
	costP90 := s.calculatePercentile(metrics.Costs, 90)

	prediction := &P90Prediction{
		TokenLimit:   tokenP90 * CONFIDENCE_FACTOR,
		MessageLimit: int(float64(messageP90) * CONFIDENCE_FACTOR),
		CostLimit:    costP90 * CONFIDENCE_FACTOR,
		Confidence:   CONFIDENCE_FACTOR,
		PredictedAt:  time.Now().UTC(),
	}

	burnRate, err := s.calculateBurnRateByProject(projectName)
	if err != nil {
		burnRate = 0
	}
	prediction.BurnRate = burnRate

	timeToLimit, err := s.calculateTimeToLimitByProject(projectName, prediction.TokenLimit, burnRate)
	if err != nil {
		timeToLimit = 0
	}
	prediction.TimeToLimit = timeToLimit

	return prediction, nil
}

// getHistoricalMetrics retrieves usage metrics from the last N hours
func (s *P90PredictionService) getHistoricalMetrics(hours int) (*UsageMetrics, error) {
	cutoffTime := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	
	query := `
		SELECT 
			sw.total_tokens,
			sw.message_count,
			COALESCE(sw.total_cost, 0) as total_cost
		FROM session_windows sw
		WHERE sw.window_start >= ?
		AND sw.total_tokens > 0
		ORDER BY sw.window_start ASC
	`
	
	rows, err := s.db.Query(query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical metrics: %w", err)
	}
	defer rows.Close()
	
	metrics := &UsageMetrics{
		Tokens:   make([]float64, 0),
		Messages: make([]int, 0),
		Costs:    make([]float64, 0),
	}
	
	for rows.Next() {
		var tokens int
		var messages int
		var cost float64
		
		err := rows.Scan(&tokens, &messages, &cost)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metrics row: %w", err)
		}
		
		metrics.Tokens = append(metrics.Tokens, float64(tokens))
		metrics.Messages = append(metrics.Messages, messages)
		metrics.Costs = append(metrics.Costs, cost)
	}
	
	return metrics, nil
}

// getHistoricalMetricsByProject retrieves usage metrics for a specific project
func (s *P90PredictionService) getHistoricalMetricsByProject(projectName string, hours int) (*UsageMetrics, error) {
	cutoffTime := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	
	query := `
		SELECT 
			sw.total_tokens,
			sw.message_count,
			COALESCE(sw.total_cost, 0) as total_cost
		FROM session_windows sw
		INNER JOIN sessions s ON s.session_window_id = sw.id
		WHERE sw.window_start >= ?
		AND s.project_name = ?
		AND sw.total_tokens > 0
		ORDER BY sw.window_start ASC
	`
	
	rows, err := s.db.Query(query, cutoffTime, projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical metrics for project: %w", err)
	}
	defer rows.Close()
	
	metrics := &UsageMetrics{
		Tokens:   make([]float64, 0),
		Messages: make([]int, 0),
		Costs:    make([]float64, 0),
	}
	
	for rows.Next() {
		var tokens int
		var messages int
		var cost float64
		
		err := rows.Scan(&tokens, &messages, &cost)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project metrics row: %w", err)
		}
		
		metrics.Tokens = append(metrics.Tokens, float64(tokens))
		metrics.Messages = append(metrics.Messages, messages)
		metrics.Costs = append(metrics.Costs, cost)
	}
	
	return metrics, nil
}

// calculatePercentile calculates the specified percentile for float64 values
func (s *P90PredictionService) calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	
	index := percentile / 100 * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	
	if lower == upper {
		return sorted[lower]
	}
	
	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// calculatePercentileInt calculates the specified percentile for int values
func (s *P90PredictionService) calculatePercentileInt(values []int, percentile float64) int {
	if len(values) == 0 {
		return 0
	}
	
	floatValues := make([]float64, len(values))
	for i, v := range values {
		floatValues[i] = float64(v)
	}
	
	return int(s.calculatePercentile(floatValues, percentile))
}

// calculateBurnRate calculates the current burn rate (tokens per hour)
func (s *P90PredictionService) calculateBurnRate() (float64, error) {
	// Get usage from the last hour
	oneHourAgo := time.Now().UTC().Add(-time.Hour)
	
	query := `
		SELECT COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens
		FROM messages
		WHERE timestamp >= ?
		AND message_role = 'assistant'
	`
	
	var tokensLastHour int
	err := s.db.QueryRow(query, oneHourAgo).Scan(&tokensLastHour)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate burn rate: %w", err)
	}
	
	return float64(tokensLastHour), nil
}

// calculateBurnRateByProject calculates burn rate for a specific project
func (s *P90PredictionService) calculateBurnRateByProject(projectName string) (float64, error) {
	oneHourAgo := time.Now().UTC().Add(-time.Hour)
	
	query := `
		SELECT COALESCE(SUM(m.input_tokens + m.output_tokens), 0) as total_tokens
		FROM messages m
		INNER JOIN sessions s ON m.session_id = s.id
		WHERE m.timestamp >= ?
		AND m.message_role = 'assistant'
		AND s.project_name = ?
	`
	
	var tokensLastHour int
	err := s.db.QueryRow(query, oneHourAgo, projectName).Scan(&tokensLastHour)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate burn rate for project: %w", err)
	}
	
	return float64(tokensLastHour), nil
}

// calculateTimeToLimit estimates minutes until reaching the limit
func (s *P90PredictionService) calculateTimeToLimit(tokenLimit float64, burnRatePerHour float64) (int, error) {
	if burnRatePerHour <= 0 {
		return 0, nil // No usage or negative usage, can't predict
	}
	
	// Get current token usage
	tokenService := NewTokenService(s.db)
	currentUsage, err := tokenService.GetCurrentTokenUsage()
	if err != nil {
		return 0, fmt.Errorf("failed to get current usage: %w", err)
	}
	
	remainingTokens := tokenLimit - float64(currentUsage.TotalTokens)
	if remainingTokens <= 0 {
		return 0, nil // Already at or over limit
	}
	
	hoursToLimit := remainingTokens / burnRatePerHour
	minutesToLimit := int(hoursToLimit * 60)
	
	return minutesToLimit, nil
}

// calculateTimeToLimitByProject estimates time to limit for a specific project
func (s *P90PredictionService) calculateTimeToLimitByProject(projectName string, tokenLimit float64, burnRatePerHour float64) (int, error) {
	if burnRatePerHour <= 0 {
		return 0, nil
	}
	
	// Get current project usage in the active window
	query := `
		SELECT COALESCE(SUM(sw.total_tokens), 0) as project_tokens
		FROM session_windows sw
		INNER JOIN sessions s ON s.session_window_id = sw.id
		WHERE sw.is_active = true
		AND s.project_name = ?
	`
	
	var currentProjectTokens int
	err := s.db.QueryRow(query, projectName).Scan(&currentProjectTokens)
	if err != nil {
		return 0, fmt.Errorf("failed to get current project usage: %w", err)
	}
	
	remainingTokens := tokenLimit - float64(currentProjectTokens)
	if remainingTokens <= 0 {
		return 0, nil
	}
	
	hoursToLimit := remainingTokens / burnRatePerHour
	minutesToLimit := int(hoursToLimit * 60)
	
	return minutesToLimit, nil
}

// GetBurnRateHistory returns burn rate history for visualization
func (s *P90PredictionService) GetBurnRateHistory(hours int) ([]models.BurnRatePoint, error) {
	cutoffTime := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	
	query := `
		SELECT 
			DATE_TRUNC('hour', m.timestamp) as hour,
			COALESCE(SUM(m.input_tokens + m.output_tokens), 0) as tokens_per_hour
		FROM messages m
		WHERE m.timestamp >= ?
		AND m.message_role = 'assistant'
		GROUP BY DATE_TRUNC('hour', m.timestamp)
		ORDER BY hour ASC
	`
	
	rows, err := s.db.Query(query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query burn rate history: %w", err)
	}
	defer rows.Close()
	
	var history []models.BurnRatePoint
	
	for rows.Next() {
		var hour time.Time
		var tokensPerHour int
		
		err := rows.Scan(&hour, &tokensPerHour)
		if err != nil {
			return nil, fmt.Errorf("failed to scan burn rate row: %w", err)
		}
		
		history = append(history, models.BurnRatePoint{
			Timestamp: hour,
			TokensPerHour: tokensPerHour,
		})
	}
	
	return history, nil
}