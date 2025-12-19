package api

import (
	"math"
	"testing"
)

// Test estimateSpotFromPutCallParity
func TestEstimateSpotFromPutCallParity(t *testing.T) {
	tests := []struct {
		name          string
		options       []OptionData
		expectedSpot  float64
		expectedCount int
		tolerance     float64
	}{
		{
			name: "single ATM option",
			options: []OptionData{
				{Strike: 100, CallMid: 10, PutMid: 10}, // S ≈ 100 + 10 - 10 = 100
			},
			expectedSpot:  100,
			expectedCount: 1,
			tolerance:     0.01,
		},
		{
			name: "multiple options with outlier",
			options: []OptionData{
				{Strike: 95, CallMid: 10, PutMid: 4},  // S ≈ 101
				{Strike: 100, CallMid: 8, PutMid: 7},  // S ≈ 101
				{Strike: 105, CallMid: 4, PutMid: 10}, // S ≈ 99
				{Strike: 110, CallMid: 20, PutMid: 1}, // S ≈ 129 (outlier)
			},
			expectedSpot:  101, // Median should ignore outlier
			expectedCount: 4,
			tolerance:     1.0,
		},
		{
			name: "no valid pairs",
			options: []OptionData{
				{Strike: 100, CallMid: 10, PutMid: 0},
				{Strike: 105, CallMid: 0, PutMid: 10},
			},
			expectedSpot:  105, // Fallback to middle strike (index 1 of 2)
			expectedCount: 0,
			tolerance:     0.5,
		},
		{
			name:          "empty options",
			options:       []OptionData{},
			expectedSpot:  0,
			expectedCount: 0,
			tolerance:     0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spot, count := estimateSpotFromPutCallParity(tt.options)
			if math.Abs(spot-tt.expectedSpot) > tt.tolerance {
				t.Errorf("spot = %f, want %f (tolerance %f)", spot, tt.expectedSpot, tt.tolerance)
			}
			if count != tt.expectedCount {
				t.Errorf("count = %d, want %d", count, tt.expectedCount)
			}
		})
	}
}

// Test calculateBidAskSpread
func TestCalculateBidAskSpread(t *testing.T) {
	tests := []struct {
		name     string
		bid      float64
		ask      float64
		mid      float64
		expected float64
	}{
		{"tight spread", 9.5, 10.5, 10.0, 0.10},         // 1 / 10 = 10%
		{"wide spread", 8.0, 12.0, 10.0, 0.40},          // 4 / 10 = 40%
		{"zero mid", 1.0, 2.0, 0.0, 0.0},                // Should handle gracefully
		{"perfect market", 10.0, 10.0, 10.0, 0.0},       // No spread
		{"penny spread", 99.49, 99.51, 99.50, 0.000201}, // 0.02 / 99.50 = 0.0201%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBidAskSpread(tt.bid, tt.ask, tt.mid)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("spread = %f, want %f", result, tt.expected)
			}
		})
	}
}

// Test isOptionLiquid
func TestIsOptionLiquid(t *testing.T) {
	tests := []struct {
		name      string
		opt       OptionData
		spotPrice float64
		minPrice  float64
		maxSpread float64
		expected  bool
	}{
		{
			name: "liquid OTM put",
			opt: OptionData{
				Strike: 90,
				PutBid: 1.9,
				PutAsk: 2.1,
				PutMid: 2.0,
			},
			spotPrice: 100,
			minPrice:  0.05,
			maxSpread: 0.20,
			expected:  true, // Spread = 0.2/2.0 = 10% < 20%
		},
		{
			name: "illiquid OTM put - wide spread",
			opt: OptionData{
				Strike: 90,
				PutBid: 1.0,
				PutAsk: 3.0,
				PutMid: 2.0,
			},
			spotPrice: 100,
			minPrice:  0.05,
			maxSpread: 0.20,
			expected:  false, // Spread = 100% > 20%
		},
		{
			name: "illiquid OTM put - price too low",
			opt: OptionData{
				Strike: 90,
				PutBid: 0.01,
				PutAsk: 0.03,
				PutMid: 0.02,
			},
			spotPrice: 100,
			minPrice:  0.05,
			maxSpread: 0.20,
			expected:  false, // Price 0.02 < 0.05
		},
		{
			name: "liquid OTM call",
			opt: OptionData{
				Strike:  110,
				CallBid: 1.9,
				CallAsk: 2.1,
				CallMid: 2.0,
			},
			spotPrice: 100,
			minPrice:  0.05,
			maxSpread: 0.20,
			expected:  true, // Spread = 0.2/2.0 = 10% < 20%
		},
		{
			name: "ATM option",
			opt: OptionData{
				Strike: 100,
				PutBid: 4.5,
				PutAsk: 5.5,
				PutMid: 5.0,
			},
			spotPrice: 100,
			minPrice:  0.05,
			maxSpread: 0.20,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOptionLiquid(tt.opt, tt.spotPrice, tt.minPrice, tt.maxSpread)
			if result != tt.expected {
				t.Errorf("isOptionLiquid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test buildCDFFromIVs
func TestBuildCDFFromIVs(t *testing.T) {
	// Create options with constant 20% IV
	options := []OptionData{
		{Strike: 80, PutIV: 0.20},
		{Strike: 90, PutIV: 0.20},
		{Strike: 100, CallIV: 0.20, PutIV: 0.20},
		{Strike: 110, CallIV: 0.20},
		{Strike: 120, CallIV: 0.20},
	}

	spotPrice := 100.0
	r := 0.04
	q := 0.012
	T := 1.0

	cdfPoints := buildCDFFromIVs(options, spotPrice, r, q, T)

	// Should have 5 valid CDF points
	if len(cdfPoints) != 5 {
		t.Fatalf("got %d CDF points, want 5", len(cdfPoints))
	}

	// Check CDF is monotonically increasing
	for i := 1; i < len(cdfPoints); i++ {
		if cdfPoints[i].CDF < cdfPoints[i-1].CDF {
			t.Errorf("CDF not monotonic: CDF(%f) = %f < CDF(%f) = %f",
				cdfPoints[i].Strike, cdfPoints[i].CDF,
				cdfPoints[i-1].Strike, cdfPoints[i-1].CDF)
		}
	}

	// CDF(80) should be low (< 50%), CDF(120) should be high (> 50%)
	if cdfPoints[0].CDF > 0.5 {
		t.Errorf("CDF(80) = %f, should be < 0.5", cdfPoints[0].CDF)
	}
	if cdfPoints[4].CDF < 0.5 {
		t.Errorf("CDF(120) = %f, should be > 0.5", cdfPoints[4].CDF)
	}

	// CDF at spot should be around 50% for symmetric distribution
	atmIdx := 2 // Strike = 100
	if math.Abs(cdfPoints[atmIdx].CDF-0.5) > 0.15 {
		t.Errorf("CDF(100) = %f, should be close to 0.5", cdfPoints[atmIdx].CDF)
	}
}

// Test buildCDFFromIVs with skew
func TestBuildCDFFromIVsWithSkew(t *testing.T) {
	// Create options with volatility skew (higher IV for OTM puts)
	options := []OptionData{
		{Strike: 80, PutIV: 0.25}, // High vol OTM put
		{Strike: 90, PutIV: 0.22},
		{Strike: 100, CallIV: 0.20, PutIV: 0.20}, // ATM
		{Strike: 110, CallIV: 0.18},
		{Strike: 120, CallIV: 0.17}, // Low vol OTM call
	}

	spotPrice := 100.0
	r := 0.04
	q := 0.012
	T := 1.0

	cdfPoints := buildCDFFromIVs(options, spotPrice, r, q, T)

	// Should have 5 valid CDF points
	if len(cdfPoints) != 5 {
		t.Fatalf("got %d CDF points, want 5", len(cdfPoints))
	}

	// CDF should still be monotonic even with skew
	for i := 1; i < len(cdfPoints); i++ {
		if cdfPoints[i].CDF < cdfPoints[i-1].CDF {
			t.Errorf("CDF not monotonic with skew")
		}
	}
}

// Test interpolateCDF
func TestInterpolateCDF(t *testing.T) {
	cdfPoints := []CDFPoint{
		{Strike: 90, CDF: 0.2},
		{Strike: 100, CDF: 0.5},
		{Strike: 110, CDF: 0.8},
	}

	tests := []struct {
		name         string
		targetStrike float64
		expectedCDF  float64
		tolerance    float64
	}{
		{"exact point", 100, 0.5, 0.001},
		{"midpoint", 95, 0.35, 0.01}, // Linear interpolation: 0.2 + 0.5*(0.5-0.2)
		{"below range", 80, 0.0, 0.001},
		{"above range", 120, 1.0, 0.001},
		{"near upper bound", 105, 0.65, 0.01}, // 0.5 + 0.5*(0.8-0.5)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interpolateCDF(cdfPoints, tt.targetStrike)
			if math.Abs(result-tt.expectedCDF) > tt.tolerance {
				t.Errorf("interpolateCDF(%f) = %f, want %f", tt.targetStrike, result, tt.expectedCDF)
			}
		})
	}
}

// Test interpolateCDF edge cases
func TestInterpolateCDFEdgeCases(t *testing.T) {
	// Empty CDF
	result := interpolateCDF([]CDFPoint{}, 100)
	if result != 0.5 {
		t.Errorf("empty CDF should return 0.5, got %f", result)
	}

	// Single point
	singlePoint := []CDFPoint{{Strike: 100, CDF: 0.5}}
	result = interpolateCDF(singlePoint, 90)
	if result != 0.0 {
		t.Errorf("below single point should return 0.0, got %f", result)
	}
	result = interpolateCDF(singlePoint, 110)
	if result != 1.0 {
		t.Errorf("above single point should return 1.0, got %f", result)
	}
}

// Test findATMOption
func TestFindATMOption(t *testing.T) {
	options := []OptionData{
		{Strike: 95, CallIV: 0.22, PutIV: 0.23},
		{Strike: 100, CallIV: 0.20, PutIV: 0.20},
		{Strike: 105, CallIV: 0.19, PutIV: 0.21},
	}

	spotPrice := 101.0

	atmOpt := findATMOption(options, spotPrice)

	// Should find strike = 100 (closest to 101)
	if atmOpt.Strike != 100 {
		t.Errorf("ATM strike = %f, want 100", atmOpt.Strike)
	}

	if atmOpt.Distance != 1.0 {
		t.Errorf("ATM distance = %f, want 1.0", atmOpt.Distance)
	}

	if !atmOpt.HasCallIV || !atmOpt.HasPutIV {
		t.Error("ATM option should have both call and put IV")
	}

	if atmOpt.CallIV != 0.20 || atmOpt.PutIV != 0.20 {
		t.Error("ATM IVs not correctly set")
	}
}

// Test findATMOption edge cases
func TestFindATMOptionEdgeCases(t *testing.T) {
	// No valid IVs
	options := []OptionData{
		{Strike: 100, CallIV: 0, PutIV: 0},
	}

	atmOpt := findATMOption(options, 100)
	if atmOpt.Strike != 0 {
		t.Error("should return zero ATM option when no valid IVs")
	}

	// Exact match
	options = []OptionData{
		{Strike: 100, CallIV: 0.20},
	}
	atmOpt = findATMOption(options, 100)
	if atmOpt.Strike != 100 || atmOpt.Distance != 0 {
		t.Error("should find exact ATM match")
	}
}

// Test findOTMOptions
func TestFindOTMOptions(t *testing.T) {
	options := []OptionData{
		{Strike: 85, PutIV: 0.25},
		{Strike: 90, PutIV: 0.22}, // ~10% OTM put
		{Strike: 95, PutIV: 0.21},
		{Strike: 100, CallIV: 0.20, PutIV: 0.20},
		{Strike: 105, CallIV: 0.19},
		{Strike: 110, CallIV: 0.18}, // ~10% OTM call
		{Strike: 115, CallIV: 0.17},
	}

	spotPrice := 100.0

	otmOpts := findOTMOptions(options, spotPrice)

	if !otmOpts.HasPut || !otmOpts.HasCall {
		t.Fatal("should find both OTM put and call")
	}

	// Should find ~90 strike for put (10% OTM)
	if math.Abs(otmOpts.PutStrike-90) > 5 {
		t.Errorf("OTM put strike = %f, want ~90", otmOpts.PutStrike)
	}

	// Should find ~110 strike for call (10% OTM)
	if math.Abs(otmOpts.CallStrike-110) > 5 {
		t.Errorf("OTM call strike = %f, want ~110", otmOpts.CallStrike)
	}

	if otmOpts.PutIV != 0.22 {
		t.Errorf("OTM put IV = %f, want 0.22", otmOpts.PutIV)
	}

	if otmOpts.CallIV != 0.18 {
		t.Errorf("OTM call IV = %f, want 0.18", otmOpts.CallIV)
	}
}

// Test findOTMOptions when no OTM options available
func TestFindOTMOptionsNoData(t *testing.T) {
	// Only ATM option
	options := []OptionData{
		{Strike: 100, CallIV: 0.20, PutIV: 0.20},
	}

	otmOpts := findOTMOptions(options, 100)

	if otmOpts.HasPut || otmOpts.HasCall {
		t.Error("should not find OTM options when none available")
	}
}

// Test median
func TestMedian(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"odd count", []float64{1, 2, 3}, 2},
		{"even count", []float64{1, 2, 3, 4}, 3}, // Takes upper middle
		{"single value", []float64{5}, 5},
		{"empty", []float64{}, 0},
		{"with outliers", []float64{1, 2, 100, 3, 4}, 100}, // Middle of [1,2,3,4,100]
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := median(tt.values)
			if result != tt.expected {
				t.Errorf("median(%v) = %f, want %f", tt.values, result, tt.expected)
			}
		})
	}
}

// Test clamp
func TestClamp(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		min      float64
		max      float64
		expected float64
	}{
		{"within range", 5, 0, 10, 5},
		{"below min", -5, 0, 10, 0},
		{"above max", 15, 0, 10, 10},
		{"at min", 0, 0, 10, 0},
		{"at max", 10, 0, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clamp(tt.value, tt.min, tt.max)
			if result != tt.expected {
				t.Errorf("clamp(%f, %f, %f) = %f, want %f",
					tt.value, tt.min, tt.max, result, tt.expected)
			}
		})
	}
}

// Integration test: full CDF workflow
func TestCDFWorkflowIntegration(t *testing.T) {
	// Create realistic options chain
	options := []OptionData{
		{Strike: 600, PutIV: 0.22},
		{Strike: 640, PutIV: 0.20},
		{Strike: 680, CallIV: 0.18, PutIV: 0.18},
		{Strike: 720, CallIV: 0.17},
		{Strike: 760, CallIV: 0.16},
	}

	spotPrice := 680.0
	r := 0.04
	q := 0.012
	T := 1.17

	// Build CDF
	cdfPoints := buildCDFFromIVs(options, spotPrice, r, q, T)

	if len(cdfPoints) != 5 {
		t.Fatalf("expected 5 CDF points, got %d", len(cdfPoints))
	}

	// Calculate probabilities for ±5% moves
	up5 := spotPrice * 1.05   // 714
	down5 := spotPrice * 0.95 // 646

	pDown5 := interpolateCDF(cdfPoints, down5)
	pUp5 := interpolateCDF(cdfPoints, up5)

	// P(loss 5%+) = P(S < 646)
	probLoss5 := pDown5

	// P(gain 5%+) = P(S > 714) = 1 - P(S < 714)
	probGain5 := 1.0 - pUp5

	// P(flat) = P(646 < S < 714)
	probFlat := pUp5 - pDown5

	// Probabilities should sum to 1
	total := probLoss5 + probGain5 + probFlat
	if math.Abs(total-1.0) > 0.01 {
		t.Errorf("probabilities sum to %f, want 1.0", total)
	}

	// Both gain and loss should be non-zero and reasonable
	if probGain5 <= 0 || probGain5 >= 1 {
		t.Errorf("prob_gain_5 = %f, should be in (0, 1)", probGain5)
	}
	if probLoss5 <= 0 || probLoss5 >= 1 {
		t.Errorf("prob_loss_5 = %f, should be in (0, 1)", probLoss5)
	}
	if probFlat <= 0 || probFlat >= 1 {
		t.Errorf("prob_flat = %f, should be in (0, 1)", probFlat)
	}
}
