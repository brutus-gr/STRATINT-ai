package api

import (
	"log/slog"
	"math"
	"os"
	"testing"
)

// Test helper: create test options handler
func newTestHandler() *OptionsAnalysisHandler {
	return &OptionsAnalysisHandler{
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})),
	}
}

// Test normCDF function
func TestNormCDF(t *testing.T) {
	tests := []struct {
		x         float64
		expected  float64
		tolerance float64
	}{
		{0.0, 0.5, 0.0001},
		{1.0, 0.8413, 0.001},
		{-1.0, 0.1587, 0.001},
		{2.0, 0.9772, 0.001},
		{-2.0, 0.0228, 0.001},
	}

	for _, tt := range tests {
		result := normCDF(tt.x)
		if math.Abs(result-tt.expected) > tt.tolerance {
			t.Errorf("normCDF(%f) = %f, want %f (tolerance %f)", tt.x, result, tt.expected, tt.tolerance)
		}
	}
}

// Test Black-Scholes call pricing
func TestBlackScholesCall(t *testing.T) {
	// ATM call with 20% vol, 1 year expiry
	S := 100.0
	K := 100.0
	T := 1.0
	r := 0.04
	q := 0.012
	sigma := 0.20

	price := blackScholesCall(S, K, T, r, q, sigma)

	// ATM call should be worth roughly S * N(0.5*sigma*sqrt(T)) + premium
	// Expected: ~$10-12 for these parameters
	if price < 8.0 || price > 15.0 {
		t.Errorf("ATM call price %f outside reasonable range [8, 15]", price)
	}

	// Deep ITM call should approach intrinsic value
	price = blackScholesCall(150.0, 100.0, 1.0, r, q, sigma)
	intrinsic := 50.0
	if price < intrinsic || price > intrinsic+10.0 {
		t.Errorf("Deep ITM call price %f should be close to intrinsic %f", price, intrinsic)
	}
}

// Test Black-Scholes put pricing
func TestBlackScholesPut(t *testing.T) {
	S := 100.0
	K := 100.0
	T := 1.0
	r := 0.04
	q := 0.012
	sigma := 0.20

	price := blackScholesPut(S, K, T, r, q, sigma)

	// ATM put should be worth roughly same as ATM call minus forward adjustment
	// Expected: ~$6-10 for these parameters
	if price < 5.0 || price > 12.0 {
		t.Errorf("ATM put price %f outside reasonable range [5, 12]", price)
	}
}

// Test implied volatility calculation converges
func TestImpliedVolatilityCall(t *testing.T) {
	S := 100.0
	K := 100.0
	T := 1.0
	r := 0.04
	q := 0.012
	targetSigma := 0.25

	// Generate target price
	targetPrice := blackScholesCall(S, K, T, r, q, targetSigma)

	// Solve for IV
	impliedSigma := impliedVolatilityCall(targetPrice, S, K, T, r, q)

	// Should recover original volatility
	if math.Abs(impliedSigma-targetSigma) > 0.001 {
		t.Errorf("IV solver: got %f, want %f", impliedSigma, targetSigma)
	}
}

// Test CDF-based probability calculation
func TestCalculateRiskNeutralProbsV2(t *testing.T) {
	h := newTestHandler()

	// Create synthetic options with constant 20% IV
	S := 100.0
	T := 1.0
	r := 0.04
	q := 0.012
	sigma := 0.20

	var options []OptionData

	// Generate strikes from 70 to 130
	for strike := 70.0; strike <= 130.0; strike += 5.0 {
		opt := OptionData{
			Strike: strike,
		}

		// Calculate theoretical prices
		if strike <= S {
			opt.PutMid = blackScholesPut(S, strike, T, r, q, sigma)
			opt.PutIV = sigma
		} else {
			opt.CallMid = blackScholesCall(S, strike, T, r, q, sigma)
			opt.CallIV = sigma
		}

		options = append(options, opt)
	}

	probs := h.calculateRiskNeutralProbsV2(options, S)

	// Check probabilities sum close to 1 (gain0+ and loss0+ should sum to 1.0)
	total := probs.ProbGain0Plus + probs.ProbLoss0Plus
	if math.Abs(total-1.0) > 0.01 {
		t.Errorf("Probabilities (gain0+ + loss0+) sum to %f, want 1.0", total)
	}

	// For lognormal distribution with 20% vol, there is slight positive skew
	// With dividend yield offsetting drift, P(gain 5%) should be close to P(loss 5%)
	ratio := probs.ProbGain5Plus / probs.ProbLoss5Plus
	if ratio < 0.9 || ratio > 1.3 {
		t.Errorf("Gain/Loss probability ratio %f should be near 1.0 (0.9-1.3) with r-q drift", ratio)
	}

	// Sanity checks
	if probs.ProbGain10Plus > probs.ProbGain5Plus {
		t.Error("P(gain 10%) should be less than P(gain 5%)")
	}
	if probs.ProbLoss10Plus > probs.ProbLoss5Plus {
		t.Error("P(loss 10%) should be less than P(loss 5%)")
	}
}

// Test expected return calculation
func TestCalculateExpectedReturnV2(t *testing.T) {
	h := newTestHandler()

	// Create synthetic options
	S := 100.0
	daysToExpiry := 365 // 1 year
	r := 0.04
	q := 0.012 // dividend yield
	sigma := 0.20

	var options []OptionData

	// Generate strikes from 50 to 150
	for strike := 50.0; strike <= 150.0; strike += 5.0 {
		opt := OptionData{
			Strike: strike,
		}

		if strike <= S {
			opt.PutIV = sigma
		} else {
			opt.CallIV = sigma
		}

		options = append(options, opt)
	}

	annualReturn := h.calculateExpectedReturnV2(options, S, daysToExpiry)

	// Should return theoretical (r - q) = 4% - 1.2% = 2.8%
	expectedReturn := (r - q) * 100.0 // 2.8%

	// Should be exact since we're returning the theoretical value
	if math.Abs(annualReturn-expectedReturn) > 0.001 {
		t.Errorf("Expected return %f%%, want %f%%",
			annualReturn, expectedReturn)
	}
}

// Test that expected return returns theoretical value
func TestExpectedReturnTheoretical(t *testing.T) {
	h := newTestHandler()

	S := 678.0
	daysToExpiry := 427

	// Doesn't matter what options we pass, should always get theoretical return
	options := []OptionData{
		{Strike: 600.0, PutIV: 0.20},
		{Strike: 700.0, CallIV: 0.18},
	}

	r := 0.04
	q := 0.012
	expectedReturn := (r - q) * 100.0 // 2.8%

	annualReturn := h.calculateExpectedReturnV2(options, S, daysToExpiry)

	if math.Abs(annualReturn-expectedReturn) > 0.001 {
		t.Errorf("Expected return %f%%, want %f%%", annualReturn, expectedReturn)
	}
}

// Test IV metrics calculation
func TestCalculateIVMetricsV2(t *testing.T) {
	h := newTestHandler()
	S := 100.0

	// Create options with skew (higher IV for OTM puts)
	options := []OptionData{
		{Strike: 80.0, PutIV: 0.25}, // OTM put - high IV
		{Strike: 90.0, PutIV: 0.22},
		{Strike: 100.0, CallIV: 0.20, PutIV: 0.20}, // ATM
		{Strike: 110.0, CallIV: 0.18},
		{Strike: 120.0, CallIV: 0.17}, // OTM call - low IV
	}

	metrics := h.calculateIVMetricsV2(options, S)

	// ATM IV should be 20%
	if math.Abs(metrics.ATMImpliedVol-20.0) > 1.0 {
		t.Errorf("ATM IV = %f%%, want 20%%", metrics.ATMImpliedVol)
	}

	// Should detect negative skew (put IV > call IV)
	if metrics.IVSkew <= 0 {
		t.Errorf("IV Skew = %f%%, should be positive (put skew)", metrics.IVSkew)
	}
}

// Test edge case: very short expiry
func TestShortExpiry(t *testing.T) {
	S := 100.0
	K := 100.0
	T := 0.01 // ~4 days
	r := 0.04
	q := 0.012
	sigma := 0.20

	price := blackScholesCall(S, K, T, r, q, sigma)

	// Very short ATM call should be small but positive
	if price <= 0 || price > 5.0 {
		t.Errorf("Short expiry ATM call price %f outside reasonable range (0, 5]", price)
	}
}

// Test edge case: zero volatility
func TestZeroVolatility(t *testing.T) {
	S := 100.0
	K := 95.0
	T := 1.0
	r := 0.04
	q := 0.012
	sigma := 0.0

	// Zero vol should give intrinsic value
	callPrice := blackScholesCall(S, K, T, r, q, sigma)
	expectedIntrinsic := S - K

	if math.Abs(callPrice-expectedIntrinsic) > 0.01 {
		t.Errorf("Zero vol call price %f, want intrinsic %f", callPrice, expectedIntrinsic)
	}
}
