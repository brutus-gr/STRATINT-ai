package api

import (
	"math"
	"sort"
)

// estimateSpotFromPutCallParity estimates spot price using put-call parity
// S ≈ K + (C - P) for each strike where both call and put exist
// Returns median to be robust to outliers
func estimateSpotFromPutCallParity(options []OptionData) (float64, int) {
	var estimates []float64

	for _, opt := range options {
		if opt.CallMid > 0 && opt.PutMid > 0 {
			// S ≈ K + (C - P)
			implied := opt.Strike + opt.CallMid - opt.PutMid
			if implied > 0 {
				estimates = append(estimates, implied)
			}
		}
	}

	if len(estimates) == 0 {
		// Fallback to middle strike
		if len(options) > 0 {
			return options[len(options)/2].Strike, 0
		}
		return 0, 0
	}

	// Use median
	sort.Float64s(estimates)
	median := estimates[len(estimates)/2]

	return median, len(estimates)
}

// calculateBidAskSpread returns bid-ask spread as fraction of mid price
func calculateBidAskSpread(bid, ask, mid float64) float64 {
	if mid <= 0 {
		return 0
	}
	return (ask - bid) / mid
}

// isOptionLiquid checks if option meets liquidity criteria
func isOptionLiquid(opt OptionData, spotPrice float64, minPrice, maxSpread float64) bool {
	var price, spread float64

	if opt.Strike <= spotPrice {
		// Below spot: check put liquidity
		price = opt.PutMid
		spread = calculateBidAskSpread(opt.PutBid, opt.PutAsk, opt.PutMid)
	} else {
		// Above spot: check call liquidity
		price = opt.CallMid
		spread = calculateBidAskSpread(opt.CallBid, opt.CallAsk, opt.CallMid)
	}

	return price >= minPrice && spread <= maxSpread
}

// CDFPoint represents a point on the cumulative distribution function
type CDFPoint struct {
	Strike float64
	CDF    float64 // P(S_T < Strike)
}

// buildCDFFromIVs constructs CDF from implied volatilities using Black-Scholes
// P(S_T < K) = N(-d2) where d1 = (ln(S/K) + (r-q+0.5*sigma^2)*T) / (sigma*sqrt(T))
//
//	d2 = d1 - sigma*sqrt(T)
func buildCDFFromIVs(options []OptionData, spotPrice, r, q, T float64) []CDFPoint {
	var cdfPoints []CDFPoint

	for _, opt := range options {
		var iv float64

		// Use put IV below spot, call IV above spot
		if opt.Strike <= spotPrice {
			iv = opt.PutIV
		} else {
			iv = opt.CallIV
		}

		if iv <= 0 {
			continue
		}

		// Calculate d1 and d2 using spot-based Black-Scholes
		d1 := (math.Log(spotPrice/opt.Strike) + (r-q+0.5*iv*iv)*T) / (iv * math.Sqrt(T))
		d2 := d1 - iv*math.Sqrt(T)

		// P(S_T < K) = N(-d2)
		cdf := normCDF(-d2)

		cdfPoints = append(cdfPoints, CDFPoint{
			Strike: opt.Strike,
			CDF:    cdf,
		})
	}

	return cdfPoints
}

// interpolateCDF linearly interpolates CDF at any strike value
func interpolateCDF(cdfPoints []CDFPoint, targetStrike float64) float64 {
	if len(cdfPoints) == 0 {
		return 0.5 // Default to 50%
	}

	// Below lowest strike
	if targetStrike < cdfPoints[0].Strike {
		return 0.0
	}

	// Above highest strike
	if targetStrike > cdfPoints[len(cdfPoints)-1].Strike {
		return 1.0
	}

	// Find bracketing strikes
	for i := 0; i < len(cdfPoints)-1; i++ {
		if targetStrike >= cdfPoints[i].Strike && targetStrike <= cdfPoints[i+1].Strike {
			// Linear interpolation
			fraction := (targetStrike - cdfPoints[i].Strike) /
				(cdfPoints[i+1].Strike - cdfPoints[i].Strike)
			return cdfPoints[i].CDF + fraction*(cdfPoints[i+1].CDF-cdfPoints[i].CDF)
		}
	}

	return 1.0
}

// ATMOption represents the at-the-money option closest to spot
type ATMOption struct {
	Strike    float64
	CallIV    float64
	PutIV     float64
	Distance  float64
	HasCallIV bool
	HasPutIV  bool
}

// findATMOption finds the option closest to current price
func findATMOption(options []OptionData, spotPrice float64) ATMOption {
	minDist := math.MaxFloat64
	var atmOpt ATMOption

	for _, opt := range options {
		dist := math.Abs(opt.Strike - spotPrice)
		if dist < minDist {
			hasCallIV := opt.CallIV > 0
			hasPutIV := opt.PutIV > 0

			if hasCallIV || hasPutIV {
				atmOpt = ATMOption{
					Strike:    opt.Strike,
					CallIV:    opt.CallIV,
					PutIV:     opt.PutIV,
					Distance:  dist,
					HasCallIV: hasCallIV,
					HasPutIV:  hasPutIV,
				}
				minDist = dist
			}
		}
	}

	return atmOpt
}

// OTMOptions represents out-of-the-money put and call options
type OTMOptions struct {
	PutStrike  float64
	PutIV      float64
	CallStrike float64
	CallIV     float64
	HasPut     bool
	HasCall    bool
}

// findOTMOptions finds OTM put (~10% below spot) and OTM call (~10% above spot)
func findOTMOptions(options []OptionData, spotPrice float64) OTMOptions {
	putTarget := spotPrice * 0.90
	callTarget := spotPrice * 1.10

	minPutDist := math.MaxFloat64
	minCallDist := math.MaxFloat64

	var result OTMOptions

	for _, opt := range options {
		// Look for OTM put
		if opt.Strike < spotPrice && opt.PutIV > 0 {
			dist := math.Abs(opt.Strike - putTarget)
			if dist < minPutDist {
				result.PutStrike = opt.Strike
				result.PutIV = opt.PutIV
				result.HasPut = true
				minPutDist = dist
			}
		}

		// Look for OTM call
		if opt.Strike > spotPrice && opt.CallIV > 0 {
			dist := math.Abs(opt.Strike - callTarget)
			if dist < minCallDist {
				result.CallStrike = opt.Strike
				result.CallIV = opt.CallIV
				result.HasCall = true
				minCallDist = dist
			}
		}
	}

	return result
}

// median returns the median of a sorted slice
func median(sorted []float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[len(sorted)/2]
}

// clamp restricts value to [min, max] range
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
