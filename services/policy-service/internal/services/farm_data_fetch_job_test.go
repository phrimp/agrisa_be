package services

import (
	"policy-service/internal/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// TEST HELPERS
// ============================================================================

func createTestMonitoringData(
	farmID uuid.UUID,
	dataSourceID uuid.UUID,
	paramName models.DataSourceParameterName,
	timestamp int64,
	value float64,
) models.FarmMonitoringData {
	unit := "index"
	return models.FarmMonitoringData{
		ID:                   uuid.New(),
		FarmID:               farmID,
		DataSourceID:         dataSourceID,
		ParameterName:        paramName,
		MeasuredValue:        value,
		MeasurementTimestamp: timestamp,
		Unit:                 &unit,
	}
}

// ============================================================================
// TEST SUITE 1: AGGREGATION FUNCTIONS
// ============================================================================

func TestApplyAggregation_Sum(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now().Unix()
	coverageStart := now - (30 * 24 * 60 * 60) // 30 days ago

	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.RainFall, now-(5*24*60*60), 10.0),
		createTestMonitoringData(farmID, dataSourceID, models.RainFall, now-(4*24*60*60), 15.0),
		createTestMonitoringData(farmID, dataSourceID, models.RainFall, now-(3*24*60*60), 20.0),
		createTestMonitoringData(farmID, dataSourceID, models.RainFall, now-(2*24*60*60), 25.0),
	}

	result := service.applyAggregation(data, models.AggregationSum, 7, coverageStart)

	assert.Equal(t, 70.0, result, "Sum should be 10+15+20+25=70")
}

func TestApplyAggregation_Average(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now().Unix()
	coverageStart := now - (30 * 24 * 60 * 60)

	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(5*24*60*60), 0.2),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(4*24*60*60), 0.4),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(3*24*60*60), 0.6),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(2*24*60*60), 0.8),
	}

	result := service.applyAggregation(data, models.AggregationAvg, 7, coverageStart)

	expected := (0.2 + 0.4 + 0.6 + 0.8) / 4.0
	assert.InDelta(t, expected, result, 0.01, "Average should be (0.2+0.4+0.6+0.8)/4=0.5")
}

func TestApplyAggregation_Min(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now().Unix()
	coverageStart := now - (30 * 24 * 60 * 60)

	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(5*24*60*60), 0.5),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(4*24*60*60), 0.2), // Min
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(3*24*60*60), 0.8),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(2*24*60*60), 0.6),
	}

	result := service.applyAggregation(data, models.AggregationMin, 7, coverageStart)

	assert.Equal(t, 0.2, result, "Minimum should be 0.2")
}

func TestApplyAggregation_Max(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now().Unix()
	coverageStart := now - (30 * 24 * 60 * 60)

	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(5*24*60*60), 0.5),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(4*24*60*60), 0.9), // Max
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(3*24*60*60), 0.3),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(2*24*60*60), 0.6),
	}

	result := service.applyAggregation(data, models.AggregationMax, 7, coverageStart)

	assert.Equal(t, 0.9, result, "Maximum should be 0.9")
}

func TestApplyAggregation_Change(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now().Unix()
	coverageStart := now - (30 * 24 * 60 * 60)

	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(5*24*60*60), 0.8), // First
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(4*24*60*60), 0.7),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(3*24*60*60), 0.5),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(2*24*60*60), 0.3), // Last
	}

	result := service.applyAggregation(data, models.AggregationChange, 7, coverageStart)

	expected := 0.3 - 0.8 // -0.5
	assert.Equal(t, expected, result, "Change should be last - first = 0.3 - 0.8 = -0.5")
}

func TestApplyAggregation_EmptyData(t *testing.T) {
	service := &RegisteredPolicyService{}
	now := time.Now().Unix()
	coverageStart := now - (30 * 24 * 60 * 60)

	data := []models.FarmMonitoringData{}

	result := service.applyAggregation(data, models.AggregationAvg, 7, coverageStart)

	assert.Equal(t, 0.0, result, "Empty data should return 0")
}

func TestApplyAggregation_WindowFiltering(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now().Unix()
	coverageStart := now - (30 * 24 * 60 * 60)

	data := []models.FarmMonitoringData{
		// Outside window (10 days ago)
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(10*24*60*60), 0.9),
		// Inside window (5 days ago)
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(5*24*60*60), 0.2),
		// Inside window (3 days ago)
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(3*24*60*60), 0.4),
	}

	// 7-day window should only include the last 2 data points
	result := service.applyAggregation(data, models.AggregationAvg, 7, coverageStart)

	expected := (0.2 + 0.4) / 2.0
	assert.InDelta(t, expected, result, 0.01, "Should only average data within 7-day window")
}

func TestApplyAggregation_CoverageStartBoundary(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now().Unix()
	coverageStart := now - (3 * 24 * 60 * 60) // Coverage started 3 days ago

	data := []models.FarmMonitoringData{
		// Before coverage start (5 days ago) - should be excluded
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(5*24*60*60), 0.9),
		// After coverage start (2 days ago) - should be included
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(2*24*60*60), 0.3),
		// After coverage start (1 day ago) - should be included
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(1*24*60*60), 0.5),
	}

	// Even with 7-day window, should only include data after coverage start
	result := service.applyAggregation(data, models.AggregationAvg, 7, coverageStart)

	expected := (0.3 + 0.5) / 2.0
	assert.InDelta(t, expected, result, 0.01, "Should only include data after coverage start")
}

// ============================================================================
// TEST SUITE 2: THRESHOLD CHECKS
// ============================================================================

func TestCheckThreshold_AllOperators(t *testing.T) {
	service := &RegisteredPolicyService{}

	tests := []struct {
		name           string
		measuredValue  float64
		thresholdValue float64
		operator       models.ThresholdOperator
		expected       bool
	}{
		// Less Than
		{"LT true", 0.2, 0.3, models.ThresholdLT, true},
		{"LT false", 0.4, 0.3, models.ThresholdLT, false},
		{"LT equal", 0.3, 0.3, models.ThresholdLT, false},

		// Less Than or Equal
		{"LTE true less", 0.2, 0.3, models.ThresholdLTE, true},
		{"LTE true equal", 0.3, 0.3, models.ThresholdLTE, true},
		{"LTE false", 0.4, 0.3, models.ThresholdLTE, false},

		// Greater Than
		{"GT true", 0.4, 0.3, models.ThresholdGT, true},
		{"GT false", 0.2, 0.3, models.ThresholdGT, false},
		{"GT equal", 0.3, 0.3, models.ThresholdGT, false},

		// Greater Than or Equal
		{"GTE true greater", 0.4, 0.3, models.ThresholdGTE, true},
		{"GTE true equal", 0.3, 0.3, models.ThresholdGTE, true},
		{"GTE false", 0.2, 0.3, models.ThresholdGTE, false},

		// Equal
		{"EQ true", 0.3, 0.3, models.ThresholdEQ, true},
		{"EQ false", 0.2, 0.3, models.ThresholdEQ, false},

		// Not Equal
		{"NE true", 0.2, 0.3, models.ThresholdNE, true},
		{"NE false", 0.3, 0.3, models.ThresholdNE, false},

		// Change Greater Than (measured value is already the change)
		{"ChangeGT true", 0.5, 0.3, models.ThresholdChangeGT, true},
		{"ChangeGT false", 0.1, 0.3, models.ThresholdChangeGT, false},

		// Change Less Than (measured value is already the change)
		{"ChangeLT true", -0.5, -0.3, models.ThresholdChangeLT, true},
		{"ChangeLT false", -0.1, -0.3, models.ThresholdChangeLT, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.checkThreshold(tt.measuredValue, tt.thresholdValue, tt.operator)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// TEST SUITE 3: LOGICAL OPERATORS
// ============================================================================

func TestEvaluateLogicalOperator(t *testing.T) {
	service := &RegisteredPolicyService{}

	tests := []struct {
		name     string
		operator models.LogicalOperator
		results  []bool
		expected bool
	}{
		// AND operator
		{"AND all true", models.LogicalAND, []bool{true, true, true}, true},
		{"AND one false", models.LogicalAND, []bool{true, false, true}, false},
		{"AND all false", models.LogicalAND, []bool{false, false, false}, false},
		{"AND empty", models.LogicalAND, []bool{}, false},
		{"AND single true", models.LogicalAND, []bool{true}, true},
		{"AND single false", models.LogicalAND, []bool{false}, false},

		// OR operator
		{"OR all true", models.LogicalOR, []bool{true, true, true}, true},
		{"OR one true", models.LogicalOR, []bool{false, true, false}, true},
		{"OR all false", models.LogicalOR, []bool{false, false, false}, false},
		{"OR empty", models.LogicalOR, []bool{}, false},
		{"OR single true", models.LogicalOR, []bool{true}, true},
		{"OR single false", models.LogicalOR, []bool{false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateLogicalOperator(tt.operator, tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// TEST SUITE 4: SATELLITE DATA GAP HANDLING
// ============================================================================

func TestAggregation_SatelliteDataGaps(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now().Unix()
	coverageStart := now - (30 * 24 * 60 * 60)

	// Simulate satellite data: 2-3 data points per week (3-4 day gaps)
	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(14*24*60*60), 0.25), // 14 days ago
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(10*24*60*60), 0.22), // 10 days ago (4-day gap)
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(7*24*60*60), 0.28),  // 7 days ago (3-day gap)
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(3*24*60*60), 0.24),  // 3 days ago (4-day gap)
	}

	tests := []struct {
		name           string
		aggFunc        models.AggregationFunction
		windowDays     int
		expectedResult float64
	}{
		{
			name:           "Mean with 14-day window (4 points)",
			aggFunc:        models.AggregationAvg,
			windowDays:     14,
			expectedResult: (0.25 + 0.22 + 0.28 + 0.24) / 4.0,
		},
		{
			name:           "Mean with 7-day window (2 points)",
			aggFunc:        models.AggregationAvg,
			windowDays:     7,
			expectedResult: (0.28 + 0.24) / 2.0,
		},
		{
			name:           "Min with 14-day window",
			aggFunc:        models.AggregationMin,
			windowDays:     14,
			expectedResult: 0.22,
		},
		{
			name:           "Max with 14-day window",
			aggFunc:        models.AggregationMax,
			windowDays:     14,
			expectedResult: 0.28,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.applyAggregation(data, tt.aggFunc, tt.windowDays, coverageStart)
			assert.InDelta(t, tt.expectedResult, result, 0.001,
				"Aggregation should handle satellite data gaps correctly")
		})
	}
}

func TestAggregation_InsufficientSatelliteData(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now().Unix()
	coverageStart := now - (30 * 24 * 60 * 60)

	// Only 1 data point in 14-day window
	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.NDVI, now-(5*24*60*60), 0.25),
	}

	// Should still work with single data point
	result := service.applyAggregation(data, models.AggregationAvg, 14, coverageStart)
	assert.Equal(t, 0.25, result, "Should handle single data point")

	// Change aggregation needs at least 2 points
	result = service.applyAggregation(data, models.AggregationChange, 14, coverageStart)
	assert.Equal(t, 0.0, result, "Change should return 0 with insufficient data")
}

// ============================================================================
// TEST SUITE 5: CONSECUTIVE DAYS (KNOWN SATELLITE LIMITATION)
// ============================================================================

func TestConsecutiveDays_WeatherData(t *testing.T) {
	// NOTE: This tests the current implementation which works for WEATHER data
	// For SATELLITE data, consecutive_required should be set to FALSE

	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now()

	// Daily weather data (no gaps)
	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -5).Unix(), 0.5), // 5 days ago
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -4).Unix(), 0.3), // 4 days ago
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -3).Unix(), 0.2), // 3 days ago
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -2).Unix(), 0.1), // 2 days ago
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -1).Unix(), 0.4), // 1 day ago
	}

	// All values < 1.0 threshold
	consecutiveDays := service.countConsecutiveDays(
		data,
		1.0,
		models.ThresholdLT,
		models.AggregationAvg,
	)

	assert.Equal(t, 5, consecutiveDays, "Should count 5 consecutive days for weather data")
}

func TestConsecutiveDays_BrokenStreak(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now()

	// Daily weather data with one day breaking the streak
	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -5).Unix(), 0.5), // < 1.0 ✓
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -4).Unix(), 0.3), // < 1.0 ✓
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -3).Unix(), 2.0), // > 1.0 ✗ BREAKS STREAK
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -2).Unix(), 0.1), // < 1.0 ✓
		createTestMonitoringData(farmID, dataSourceID, models.RainFall,
			now.AddDate(0, 0, -1).Unix(), 0.4), // < 1.0 ✓
	}

	consecutiveDays := service.countConsecutiveDays(
		data,
		1.0,
		models.ThresholdLT,
		models.AggregationAvg,
	)

	// Should only count from most recent: 2 days
	assert.Equal(t, 2, consecutiveDays, "Should only count consecutive days from most recent")
}

func TestConsecutiveDays_SatelliteDataFails(t *testing.T) {
	// IMPORTANT: This test DOCUMENTS the known limitation
	// For satellite data with 3-4 day gaps, consecutive check will FAIL
	// Solution: Set consecutive_required = false for satellite triggers

	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now()

	// Satellite data with 3-4 day gaps (realistic)
	data := []models.FarmMonitoringData{
		createTestMonitoringData(farmID, dataSourceID, models.NDVI,
			now.AddDate(0, 0, -10).Unix(), 0.25), // 10 days ago
		createTestMonitoringData(farmID, dataSourceID, models.NDVI,
			now.AddDate(0, 0, -7).Unix(), 0.22), // 7 days ago (3-day gap)
		createTestMonitoringData(farmID, dataSourceID, models.NDVI,
			now.AddDate(0, 0, -3).Unix(), 0.28), // 3 days ago (4-day gap) ← BREAKS HERE
	}

	consecutiveDays := service.countConsecutiveDays(
		data,
		0.3,
		models.ThresholdLT,
		models.AggregationAvg,
	)

	// Current code allows max 48-hour gap (2 days)
	// 4-day gap breaks the consecutive check
	assert.LessOrEqual(t, consecutiveDays, 1,
		"KNOWN LIMITATION: Satellite data gaps break consecutive check")

	// This is WHY satellite triggers must have consecutive_required = false
}

// ============================================================================
// TEST SUITE 6: EDGE CASES & ERROR HANDLING
// ============================================================================

func TestMergeMonitoringData_NoDuplicates(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()

	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	fetched := []models.FarmMonitoringData{
		{ID: id1, FarmID: farmID, DataSourceID: dataSourceID, MeasuredValue: 0.5},
		{ID: id2, FarmID: farmID, DataSourceID: dataSourceID, MeasuredValue: 0.6},
	}

	historical := []models.FarmMonitoringData{
		{ID: id2, FarmID: farmID, DataSourceID: dataSourceID, MeasuredValue: 0.6}, // Duplicate
		{ID: id3, FarmID: farmID, DataSourceID: dataSourceID, MeasuredValue: 0.7}, // New
	}

	merged := service.mergeMonitoringData(fetched, historical)

	assert.Len(t, merged, 3, "Should have 3 unique records")

	// Check all IDs present
	ids := make(map[uuid.UUID]bool)
	for _, d := range merged {
		ids[d.ID] = true
	}
	assert.True(t, ids[id1])
	assert.True(t, ids[id2])
	assert.True(t, ids[id3])
}

func TestMergeMonitoringData_NilHistorical(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()

	fetched := []models.FarmMonitoringData{
		{ID: uuid.New(), FarmID: farmID, DataSourceID: dataSourceID, MeasuredValue: 0.5},
	}

	merged := service.mergeMonitoringData(fetched, nil)

	assert.Equal(t, fetched, merged, "Should return fetched data when historical is nil")
}

func TestIsInBlackoutPeriod(t *testing.T) {
	service := &RegisteredPolicyService{}

	tests := []struct {
		name            string
		currentDate     string
		blackoutPeriods map[string]any
		expected        bool
	}{
		{
			name:            "No blackout periods",
			currentDate:     "06-15",
			blackoutPeriods: nil,
			expected:        false,
		},
		{
			name:        "Within blackout period",
			currentDate: "12-15",
			blackoutPeriods: map[string]any{
				"periods": []any{
					map[string]any{"start": "12-01", "end": "12-31"},
				},
			},
			expected: true,
		},
		{
			name:        "Outside blackout period",
			currentDate: "06-15",
			blackoutPeriods: map[string]any{
				"periods": []any{
					map[string]any{"start": "12-01", "end": "12-31"},
				},
			},
			expected: false,
		},
		{
			name:        "Wrapping blackout period (winter)",
			currentDate: "01-15",
			blackoutPeriods: map[string]any{
				"periods": []any{
					map[string]any{"start": "11-01", "end": "02-28"},
				},
			},
			expected: true,
		},
		{
			name:        "Multiple blackout periods",
			currentDate: "07-15",
			blackoutPeriods: map[string]any{
				"periods": []any{
					map[string]any{"start": "01-01", "end": "01-31"},
					map[string]any{"start": "07-01", "end": "07-31"},
					map[string]any{"start": "12-01", "end": "12-31"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse current date in format MM-DD
			currentTime, _ := time.Parse("01-02", tt.currentDate)

			result := service.isInBlackoutPeriod(tt.blackoutPeriods, currentTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateBaseline(t *testing.T) {
	service := &RegisteredPolicyService{}
	farmID := uuid.New()
	dataSourceID := uuid.New()
	now := time.Now()

	// Historical baseline data (30-37 days ago)
	// Current aggregation window (last 7 days)
	// Baseline should use data from 37-7 = 30 days ago, up to 7 days ago

	data := []models.FarmMonitoringData{
		// Baseline period (30-37 days ago)
		createTestMonitoringData(farmID, dataSourceID, models.NDVI,
			now.AddDate(0, 0, -35).Unix(), 0.8),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI,
			now.AddDate(0, 0, -32).Unix(), 0.7),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI,
			now.AddDate(0, 0, -30).Unix(), 0.9),

		// Current period (last 7 days) - should NOT be included in baseline
		createTestMonitoringData(farmID, dataSourceID, models.NDVI,
			now.AddDate(0, 0, -5).Unix(), 0.2),
		createTestMonitoringData(farmID, dataSourceID, models.NDVI,
			now.AddDate(0, 0, -3).Unix(), 0.3),
	}

	baseline := service.calculateBaseline(data, 30, models.AggregationAvg, 7)

	// Should average only the baseline period values: (0.8 + 0.7 + 0.9) / 3
	expected := (0.8 + 0.7 + 0.9) / 3.0
	assert.InDelta(t, expected, baseline, 0.01, "Baseline should not include current aggregation window")
}

// ============================================================================
// TEST SUITE 7: INTEGRATION TEST - FULL TRIGGER EVALUATION
// ============================================================================

func TestTriggerEvaluation_IntegrationTest(t *testing.T) {
	t.Skip("Integration test - requires full setup with mocked repositories")
	// This test would require proper setup of all dependencies
	// For now, we've tested all the core logic functions individually
}
