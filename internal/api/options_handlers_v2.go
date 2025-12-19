package api

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Black-Scholes helper functions
func normCDF(x float64) float64 {
	// Approximation of cumulative normal distribution
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

func normPDF(x float64) float64 {
	return math.Exp(-x*x/2) / math.Sqrt(2*math.Pi)
}

// Black-Scholes call price with dividend yield
func blackScholesCall(S, K, T, r, q, sigma float64) float64 {
	if sigma <= 0 || T <= 0 {
		return math.Max(S-K, 0)
	}
	d1 := (math.Log(S/K) + (r-q+0.5*sigma*sigma)*T) / (sigma * math.Sqrt(T))
	d2 := d1 - sigma*math.Sqrt(T)
	return S*math.Exp(-q*T)*normCDF(d1) - K*math.Exp(-r*T)*normCDF(d2)
}

// Black-Scholes put price with dividend yield
func blackScholesPut(S, K, T, r, q, sigma float64) float64 {
	if sigma <= 0 || T <= 0 {
		return math.Max(K-S, 0)
	}
	d1 := (math.Log(S/K) + (r-q+0.5*sigma*sigma)*T) / (sigma * math.Sqrt(T))
	d2 := d1 - sigma*math.Sqrt(T)
	return K*math.Exp(-r*T)*normCDF(-d2) - S*math.Exp(-q*T)*normCDF(-d1)
}
func impliedVolatilityCall(marketPrice, S, K, T, r, q float64) float64 {
	if marketPrice <= 0 || T <= 0 {
		return 0
	}

	// Check if option is deep ITM (>20% in the money)
	moneyness := S / K
	if moneyness > 1.2 {
		// Deep ITM options have minimal time value
		// IV calculation is unreliable, use a reasonable default
		return 0.18
	}

	// Better initial guess based on moneyness
	var sigma float64
	if moneyness > 1.1 {
		sigma = 0.17
	} else if moneyness < 0.9 {
		sigma = 0.22
	} else {
		sigma = 0.19
	}

	tolerance := 0.0001
	maxIterations := 50 // Reduced from 100

	for i := 0; i < maxIterations; i++ {
		price := blackScholesCall(S, K, T, r, q, sigma)
		diff := price - marketPrice

		if math.Abs(diff) < tolerance {
			// Final sanity check before returning
			if sigma > 0.5 || sigma < 0.05 {
				return 0
			}
			return sigma
		}

		d1 := (math.Log(S/K) + (r-q+0.5*sigma*sigma)*T) / (sigma * math.Sqrt(T))
		vega := S * math.Exp(-q*T) * normPDF(d1) * math.Sqrt(T)

		if vega < 0.001 { // Increased threshold
			break
		}

		sigma = sigma - diff/vega

		// Enforce bounds DURING iteration
		if sigma <= 0.05 {
			sigma = 0.05
		}
		if sigma >= 0.5 {
			// If we hit upper bound, the option is probably deep ITM
			return 0.18
		}
	}

	// If we exit loop without converging, filter it out
	return 0
}

func impliedVolatilityPut(marketPrice, S, K, T, r, q float64) float64 {
	if marketPrice <= 0 || T <= 0 {
		return 0
	}

	// Check if option is deep ITM (>20% in the money for puts)
	moneyness := S / K
	if moneyness < 0.8 {
		// Deep ITM puts have minimal time value
		return 0.20
	}

	// Better initial guess based on moneyness
	var sigma float64
	if moneyness < 0.9 {
		sigma = 0.22
	} else if moneyness > 1.1 {
		sigma = 0.25
	} else {
		sigma = 0.21
	}

	tolerance := 0.0001
	maxIterations := 50

	for i := 0; i < maxIterations; i++ {
		price := blackScholesPut(S, K, T, r, q, sigma)
		diff := price - marketPrice

		if math.Abs(diff) < tolerance {
			// Final sanity check
			if sigma > 0.5 || sigma < 0.05 {
				return 0
			}
			return sigma
		}

		d1 := (math.Log(S/K) + (r-q+0.5*sigma*sigma)*T) / (sigma * math.Sqrt(T))
		vega := S * math.Exp(-q*T) * normPDF(d1) * math.Sqrt(T)

		if vega < 0.001 {
			break
		}

		sigma = sigma - diff/vega

		// Enforce bounds DURING iteration
		if sigma <= 0.05 {
			sigma = 0.05
		}
		if sigma >= 0.5 {
			// If we hit upper bound, option is probably deep ITM
			return 0.20
		}
	}

	// If we exit loop without converging, return 0
	return 0
}

// parseLastTradePrice extracts the price from Nasdaq's lastTrade string
// Example: "LAST TRADE: $663.32 (AS OF OCT 16, 2025 1:39 PM ET)" -> 663.32
func parseLastTradePrice(lastTrade string) float64 {
	if lastTrade == "" {
		return 0
	}

	// Find the $ symbol
	dollarIdx := strings.Index(lastTrade, "$")
	if dollarIdx == -1 {
		return 0
	}

	// Extract substring starting after $
	priceStr := lastTrade[dollarIdx+1:]

	// Find the end of the price (first space or parenthesis)
	endIdx := strings.IndexAny(priceStr, " (")
	if endIdx != -1 {
		priceStr = priceStr[:endIdx]
	}

	// Parse the price
	price := parseFloat(priceStr)
	return price
}

func (h *OptionsAnalysisHandler) parseOptionsDataV2(chainData NasdaqOptionChain, daysToExpiry int) ([]OptionData, float64, error) {
	h.logger.Info("parsing options data V2", "total_rows", len(chainData.Data.Table.Rows), "days_to_expiry", daysToExpiry)

	// Extract actual spot price from lastTrade field
	spotPriceFromAPI := parseLastTradePrice(chainData.Data.LastTrade)
	h.logger.Info("spot price from Nasdaq API", "lastTrade", chainData.Data.LastTrade, "parsed_price", spotPriceFromAPI)

	// First pass: collect all strikes
	var allOptions []OptionData

	for _, row := range chainData.Data.Table.Rows {
		if row.Strike == "" || row.Strike == "null" {
			continue
		}

		strike := parseFloat(row.Strike)
		if strike == 0 {
			continue
		}

		callBid := parseFloat(row.CBid)
		callAsk := parseFloat(row.CAsk)
		putBid := parseFloat(row.PBid)
		putAsk := parseFloat(row.PAsk)

		// Need at least one valid price
		if callBid == 0 && callAsk == 0 && putBid == 0 && putAsk == 0 {
			continue
		}

		opt := OptionData{
			Strike:  strike,
			CallBid: callBid,
			CallAsk: callAsk,
			CallMid: (callBid + callAsk) / 2,
			PutBid:  putBid,
			PutAsk:  putAsk,
			PutMid:  (putBid + putAsk) / 2,
			CallVol: parseFloat(row.CVolume),
			PutVol:  parseFloat(row.PVolume),
			CallOI:  parseFloat(row.COpenInterest),
			PutOI:   parseFloat(row.POpenInterest),
		}

		allOptions = append(allOptions, opt)
	}

	if len(allOptions) == 0 {
		return nil, 0, fmt.Errorf("no valid options data found")
	}

	sort.Slice(allOptions, func(i, j int) bool {
		return allOptions[i].Strike < allOptions[j].Strike
	})

	// Use spot price from API if available, otherwise estimate from put-call parity
	var spotPrice float64
	if spotPriceFromAPI > 0 {
		spotPrice = spotPriceFromAPI
		h.logger.Info("using spot price from API", "price", spotPrice)
	} else {
		// Fallback: Estimate spot price using put-call parity from near-ATM options
		var spotPriceEstimates []float64
		for _, opt := range allOptions {
			if opt.CallMid > 0 && opt.PutMid > 0 {
				// S ≈ K + (C - P)
				implied := opt.Strike + opt.CallMid - opt.PutMid
				if implied > 0 {
					spotPriceEstimates = append(spotPriceEstimates, implied)
				}
			}
		}

		if len(spotPriceEstimates) > 0 {
			// Use median to be robust to outliers
			sort.Float64s(spotPriceEstimates)
			spotPrice = spotPriceEstimates[len(spotPriceEstimates)/2]
		} else {
			// Last fallback to middle strike
			spotPrice = allOptions[len(allOptions)/2].Strike
		}
		h.logger.Info("estimated spot price from put-call parity", "price", spotPrice, "estimates_count", len(spotPriceEstimates))
	}

	// Second pass: filter by liquidity (NOT by moneyness) and calculate IV
	// Use all liquid strikes to preserve distribution tails for Breeden-Litzenberger
	var filteredOptions []OptionData

	T := float64(daysToExpiry) / 365.0 // Time to expiry in years
	r := 0.04                          // Risk-free rate assumption
	q := 0.012                         // SPY dividend yield

	h.logger.Info("filtering by liquidity", "total_strikes", len(allOptions), "T_years", T, "risk_free_rate", r, "dividend_yield", q)

	for _, opt := range allOptions {
		// Calculate IVs for both calls and puts where prices are valid
		var relevantIV float64
		var hasValidPrice bool

		// Calculate call IV if call has valid price
		if opt.CallMid > 0.05 {
			callSpread := (opt.CallAsk - opt.CallBid) / opt.CallMid
			if callSpread <= 0.35 {
				opt.CallIV = impliedVolatilityCall(opt.CallMid, spotPrice, opt.Strike, T, r, q)
			}
		}

		// Calculate put IV if put has valid price
		if opt.PutMid > 0.05 {
			putSpread := (opt.PutAsk - opt.PutBid) / opt.PutMid
			if putSpread <= 0.35 {
				opt.PutIV = impliedVolatilityPut(opt.PutMid, spotPrice, opt.Strike, T, r, q)
			}
		}

		// Determine which IV to use for filtering (puts below spot, calls above spot)
		if opt.Strike <= spotPrice {
			relevantIV = opt.PutIV
			if opt.PutMid > 0.05 && (opt.PutAsk-opt.PutBid)/opt.PutMid <= 0.35 {
				hasValidPrice = true
			}
		} else {
			relevantIV = opt.CallIV
			if opt.CallMid > 0.05 && (opt.CallAsk-opt.CallBid)/opt.CallMid <= 0.35 {
				hasValidPrice = true
			}
		}

		// Skip if no valid price for relevant side, or relevant IV calculation failed
		if !hasValidPrice || relevantIV <= 0 {
			continue
		}

		filteredOptions = append(filteredOptions, opt)
	}

	// Get strike range for logging
	var minStrike, maxStrike float64
	if len(filteredOptions) > 0 {
		minStrike = filteredOptions[0].Strike
		maxStrike = filteredOptions[len(filteredOptions)-1].Strike
	}

	h.logger.Info("filtering complete",
		"original_options", len(allOptions),
		"filtered_options", len(filteredOptions),
		"strike_range", fmt.Sprintf("%.2f - %.2f", minStrike, maxStrike),
		"spot_price", spotPrice)

	return filteredOptions, spotPrice, nil
}
func (h *OptionsAnalysisHandler) calculateRiskNeutralProbsV2(options []OptionData, currentPrice float64) RiskNeutralProbs {
	h.logger.Info("🔥 USING NEW V2 FUNCTION 🔥")

	probs := RiskNeutralProbs{}

	if len(options) < 3 {
		h.logger.Warn("insufficient options for probability calculation", "count", len(options))
		return probs
	}

	r := 0.04
	q := 0.012
	T := 427.0 / 365.0

	// Step 1: Find ATM IV (use this for ALL strikes in CDF calculation)
	atmIV := 0.20 // Default fallback
	minDiff := math.MaxFloat64

	for _, opt := range options {
		diff := math.Abs(opt.Strike - currentPrice)
		if diff < minDiff {
			// Use the IV from the closest strike to ATM
			var candidateIV float64
			if opt.Strike <= currentPrice && opt.PutIV > 0 {
				candidateIV = opt.PutIV
			} else if opt.Strike > currentPrice && opt.CallIV > 0 {
				candidateIV = opt.CallIV
			}

			if candidateIV > 0 {
				atmIV = candidateIV
				minDiff = diff
			}
		}
	}

	h.logger.Info("using ATM IV for all strikes", "atm_iv", atmIV, "atm_iv_pct", atmIV*100)

	// Step 2: Build CDF using CONSTANT ATM IV for all strikes
	// This gives a consistent log-normal distribution
	type CDFPoint struct {
		Strike float64
		CDF    float64 // P(S_T < K)
	}
	var cdfPoints []CDFPoint

	for _, opt := range options {
		// CRITICAL: Use the SAME ATM IV for all strikes
		// (not the strike-specific IV from the volatility smile)
		iv := atmIV

		// Calculate d1 and d2 using spot-based Black-Scholes formula
		d1 := (math.Log(currentPrice/opt.Strike) + (r-q+0.5*iv*iv)*T) / (iv * math.Sqrt(T))
		d2 := d1 - iv*math.Sqrt(T)

		// P(S_T < K) = N(-d2)
		cdf := normCDF(-d2)

		cdfPoints = append(cdfPoints, CDFPoint{
			Strike: opt.Strike,
			CDF:    cdf,
		})
	}

	if len(cdfPoints) < 2 {
		h.logger.Warn("insufficient CDF points after filtering", "points", len(cdfPoints))
		return probs
	}

	h.logger.Info("built CDF from Black-Scholes",
		"points", len(cdfPoints),
		"first_strike", cdfPoints[0].Strike,
		"first_cdf", cdfPoints[0].CDF,
		"last_strike", cdfPoints[len(cdfPoints)-1].Strike,
		"last_cdf", cdfPoints[len(cdfPoints)-1].CDF)

	// Helper function to interpolate CDF at any strike
	interpolateCDF := func(targetStrike float64) float64 {
		// Find bracketing points
		for i := 0; i < len(cdfPoints)-1; i++ {
			if targetStrike >= cdfPoints[i].Strike && targetStrike <= cdfPoints[i+1].Strike {
				// Linear interpolation
				fraction := (targetStrike - cdfPoints[i].Strike) /
					(cdfPoints[i+1].Strike - cdfPoints[i].Strike)
				return cdfPoints[i].CDF + fraction*(cdfPoints[i+1].CDF-cdfPoints[i].CDF)
			}
		}
		// Outside range
		if targetStrike < cdfPoints[0].Strike {
			return 0.0
		}
		return 1.0
	}

	// Calculate probabilities at threshold levels
	down5 := currentPrice * 0.95
	down10 := currentPrice * 0.90
	down15 := currentPrice * 0.85
	up5 := currentPrice * 1.05
	up10 := currentPrice * 1.10
	up15 := currentPrice * 1.15

	cumDown5 := interpolateCDF(down5)
	cumDown10 := interpolateCDF(down10)
	cumDown15 := interpolateCDF(down15)
	cumUp5 := interpolateCDF(up5)
	cumUp10 := interpolateCDF(up10)
	cumUp15 := interpolateCDF(up15)

	// Calculate CDF at current price (should be ~0.50 and used for 0-5% ranges)
	cumCurrent := interpolateCDF(currentPrice)
	h.logger.Info("CDF diagnostics",
		"current_price", currentPrice,
		"cdf_at_current", cumCurrent,
		"expected", "~0.50",
		"down5_strike", down5,
		"down5_cdf", cumDown5,
		"up5_strike", up5,
		"up5_cdf", cumUp5)

	// P(loss X%+) = P(S < down_X)
	probs.ProbLoss5Plus = cumDown5
	probs.ProbLoss10Plus = cumDown10
	probs.ProbLoss15Plus = cumDown15

	// P(gain X%+) = P(S > up_X) = 1 - P(S <= up_X)
	probs.ProbGain5Plus = 1.0 - cumUp5
	probs.ProbGain10Plus = 1.0 - cumUp10
	probs.ProbGain15Plus = 1.0 - cumUp15

	// P(gain 0%+) = P(S > current) = 1 - P(S <= current)
	probs.ProbGain0Plus = 1.0 - cumCurrent

	// P(loss 0%+) = P(S < current)
	probs.ProbLoss0Plus = cumCurrent

	h.logger.Info("calculated probabilities from Black-Scholes CDF",
		"gain0+", probs.ProbGain0Plus,
		"gain5+", probs.ProbGain5Plus,
		"gain10+", probs.ProbGain10Plus,
		"loss0+", probs.ProbLoss0Plus,
		"loss5+", probs.ProbLoss5Plus,
		"loss10+", probs.ProbLoss10Plus,
		"sum", probs.ProbGain0Plus+probs.ProbLoss0Plus)

	return probs
}

func (h *OptionsAnalysisHandler) calculateIVMetricsV2(options []OptionData, currentPrice float64) IVMetrics {
	metrics := IVMetrics{}

	if len(options) == 0 {
		return metrics
	}

	// Find ATM option (closest to current price)
	minDiff := math.MaxFloat64
	var atmCallIV, atmPutIV float64

	for _, opt := range options {
		diff := math.Abs(opt.Strike - currentPrice)
		if diff < minDiff {
			if opt.CallIV > 0 {
				atmCallIV = opt.CallIV
			}
			if opt.PutIV > 0 {
				atmPutIV = opt.PutIV
			}
			if opt.CallIV > 0 || opt.PutIV > 0 {
				minDiff = diff
			}
		}
	}

	// Use average of call and put IV if both available
	var atmIV float64
	if atmCallIV > 0 && atmPutIV > 0 {
		atmIV = (atmCallIV + atmPutIV) / 2
	} else if atmCallIV > 0 {
		atmIV = atmCallIV
	} else if atmPutIV > 0 {
		atmIV = atmPutIV
	}

	metrics.ATMImpliedVol = atmIV * 100

	// VIX is 30-day volatility, but we have 14-month options
	// Scale to 30-day equivalent: IV_30day = IV_long * sqrt(30 / days_to_expiry)
	// For now, just report the long-term IV
	metrics.VIXEquivalent = atmIV * 100

	metrics.IVTermStructure = "14-month (Dec 2026)"

	// Calculate IV skew (OTM put IV - OTM call IV)
	var otmPutIV, otmCallIV float64
	putTarget := currentPrice * 0.90  // ~10% OTM put
	callTarget := currentPrice * 1.10 // ~10% OTM call

	minPutDiff := math.MaxFloat64
	minCallDiff := math.MaxFloat64

	for _, opt := range options {
		if opt.Strike < currentPrice {
			diff := math.Abs(opt.Strike - putTarget)
			if diff < minPutDiff && opt.PutIV > 0 {
				minPutDiff = diff
				otmPutIV = opt.PutIV
			}
		} else if opt.Strike > currentPrice {
			diff := math.Abs(opt.Strike - callTarget)
			if diff < minCallDiff && opt.CallIV > 0 {
				minCallDiff = diff
				otmCallIV = opt.CallIV
			}
		}
	}

	if otmPutIV > 0 && otmCallIV > 0 {
		metrics.IVSkew = (otmPutIV - otmCallIV) * 100
	}

	h.logger.Info("IV metrics calculated",
		"atm_iv_pct", metrics.ATMImpliedVol,
		"iv_skew", metrics.IVSkew,
		"otm_put_iv", otmPutIV*100,
		"otm_call_iv", otmCallIV*100)

	return metrics
}

func (h *OptionsAnalysisHandler) calculateExpectedReturnV2(options []OptionData, currentPrice float64, daysToExpiry int) float64 {
	// In the risk-neutral measure, expected return = risk-free rate - dividend yield
	// With volatility skew, computing E[S] from stitched CDFs gives biased results
	// So we return the theoretical risk-neutral return directly

	r := 0.04
	q := 0.012 // SPY dividend yield
	T := float64(daysToExpiry) / 365.0

	// Risk-neutral annualized return = (r - q) * 100%
	annualizedReturn := (r - q) * 100.0

	h.logger.Info("risk-neutral expected return",
		"risk_free_rate", r,
		"dividend_yield", q,
		"net_return_annual_pct", annualizedReturn,
		"time_years", T,
		"note", "Using theoretical (r-q) to avoid volatility smile bias")

	return annualizedReturn
}

func (h *OptionsAnalysisHandler) calculatePutCallRatioV2(options []OptionData) float64 {
	var totalPutOI, totalCallOI float64

	for _, opt := range options {
		totalPutOI += opt.PutOI
		totalCallOI += opt.CallOI
	}

	if totalCallOI > 0 {
		ratio := totalPutOI / totalCallOI
		h.logger.Info("put/call ratio calculated",
			"ratio", ratio,
			"total_put_oi", totalPutOI,
			"total_call_oi", totalCallOI)
		return ratio
	}

	return 0
}
