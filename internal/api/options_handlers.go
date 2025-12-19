package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"log/slog"
)

// OptionsAnalysisHandler handles GET /api/market/spy-risk-analysis
type OptionsAnalysisHandler struct {
	logger *slog.Logger
}

func NewOptionsAnalysisHandler(logger *slog.Logger) *OptionsAnalysisHandler {
	return &OptionsAnalysisHandler{
		logger: logger,
	}
}

// NasdaqOptionChain represents the response from Nasdaq API
type NasdaqOptionChain struct {
	Data struct {
		TotalRecord int    `json:"totalRecord"`
		LastTrade   string `json:"lastTrade"`
		Table       struct {
			Headers map[string]string `json:"headers"`
			Rows    []struct {
				ExpiryGroup string `json:"expirygroup"`
				ExpiryDate  string `json:"expiryDate"`
				Strike      string `json:"strike"`
				// Call data
				CLast         string `json:"c_Last"`
				CChange       string `json:"c_Change"`
				CBid          string `json:"c_Bid"`
				CAsk          string `json:"c_Ask"`
				CVolume       string `json:"c_Volume"`
				COpenInterest string `json:"c_Openinterest"`
				// Put data
				PLast         string `json:"p_Last"`
				PChange       string `json:"p_Change"`
				PBid          string `json:"p_Bid"`
				PAsk          string `json:"p_Ask"`
				PVolume       string `json:"p_Volume"`
				POpenInterest string `json:"p_Openinterest"`
			} `json:"rows"`
		} `json:"table"`
	} `json:"data"`
	Status struct {
		RCode            int    `json:"rCode"`
		BCodeMessage     string `json:"bCodeMessage"`
		DeveloperMessage string `json:"developerMessage"`
	} `json:"status"`
}

// OptionData represents a single option contract
type OptionData struct {
	Strike  float64 `json:"strike"`
	CallBid float64 `json:"call_bid"`
	CallAsk float64 `json:"call_ask"`
	CallMid float64 `json:"call_mid"`
	PutBid  float64 `json:"put_bid"`
	PutAsk  float64 `json:"put_ask"`
	PutMid  float64 `json:"put_mid"`
	CallIV  float64 `json:"call_iv"`
	PutIV   float64 `json:"put_iv"`
	CallVol float64 `json:"call_volume"`
	PutVol  float64 `json:"put_volume"`
	CallOI  float64 `json:"call_open_interest"`
	PutOI   float64 `json:"put_open_interest"`
}

// RiskAnalysisResponse is the JSON response
type RiskAnalysisResponse struct {
	Timestamp                string           `json:"timestamp"`
	Symbol                   string           `json:"symbol"`
	CurrentPrice             float64          `json:"current_price"`
	DaysToExpiry             int              `json:"days_to_expiry"`
	RiskNeutralProbabilities RiskNeutralProbs `json:"risk_neutral_probabilities"`
	ImpliedVolatilityMetrics IVMetrics        `json:"implied_volatility_metrics"`
	MarketExpectedReturn     float64          `json:"market_expected_return_percent"`
	TailRiskMetrics          TailRisk         `json:"tail_risk_metrics"`
	SkewMetrics              SkewMetrics      `json:"skew_metrics"`
	PutCallRatio             float64          `json:"put_call_ratio"`
	DataQuality              DataQuality      `json:"data_quality"`
}

type RiskNeutralProbs struct {
	ProbGain0Plus  float64 `json:"prob_gain_0pct_plus"`
	ProbGain5Plus  float64 `json:"prob_gain_5pct_plus"`
	ProbGain10Plus float64 `json:"prob_gain_10pct_plus"`
	ProbGain15Plus float64 `json:"prob_gain_15pct_plus"`
	ProbLoss0Plus  float64 `json:"prob_loss_0pct_plus"`
	ProbLoss5Plus  float64 `json:"prob_loss_5pct_plus"`
	ProbLoss10Plus float64 `json:"prob_loss_10pct_plus"`
	ProbLoss15Plus float64 `json:"prob_loss_15pct_plus"`
}

type IVMetrics struct {
	ATMImpliedVol   float64 `json:"atm_implied_vol_percent"`
	IVSkew          float64 `json:"iv_skew"`
	IVTermStructure string  `json:"iv_term_structure"`
	VIXEquivalent   float64 `json:"vix_equivalent_percent"`
}

type TailRisk struct {
	LeftTailRisk      float64 `json:"left_tail_risk_5pct"`
	RightTailRisk     float64 `json:"right_tail_risk_95pct"`
	ExpectedShortfall float64 `json:"expected_shortfall_percent"`
	KurtosisProxy     float64 `json:"kurtosis_proxy"`
}

type SkewMetrics struct {
	RiskReversalSkew float64 `json:"risk_reversal_25delta"`
	ButterflySpread  float64 `json:"butterfly_spread"`
	SkewnessEstimate float64 `json:"skewness_estimate"`
}

type DataQuality struct {
	OptionsAnalyzed int      `json:"options_analyzed"`
	StrikeRange     string   `json:"strike_range"`
	AvgBidAskSpread float64  `json:"avg_bid_ask_spread_percent"`
	Warnings        []string `json:"warnings"`
}

func (h *OptionsAnalysisHandler) HandleSPYRiskAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("fetching SPY options chain for risk analysis")

	// Use December 2026 third Friday (standard monthly expiration)
	// This is a known date with high liquidity
	expiryDate := "2026-12-18"

	h.logger.Info("using expiry date", "date", expiryDate)

	// Fetch options data from Nasdaq (limit=200 to get full strike range around current price)
	nasdaqURL := fmt.Sprintf("https://api.nasdaq.com/api/quote/SPY/option-chain?assetclass=etf&limit=200&fromdate=%s&todate=%s&excode=oprac&callput=callput&money=all&type=all",
		expiryDate, expiryDate)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", nasdaqURL, nil)
	if err != nil {
		h.logger.Error("failed to create nasdaq request", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set comprehensive headers to mimic real browser
	// Note: Don't set Accept-Encoding manually - let Go handle compression automatically
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.nasdaq.com/")
	req.Header.Set("Origin", "https://www.nasdaq.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	h.logger.Info("requesting nasdaq data", "url", nasdaqURL)

	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error("failed to fetch nasdaq data", "error", err)
		http.Error(w, "Failed to fetch market data", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	h.logger.Info("nasdaq response received", "status", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"))

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("nasdaq api returned non-200", "status", resp.StatusCode)
		http.Error(w, fmt.Sprintf("Market data unavailable (HTTP %d)", resp.StatusCode), http.StatusServiceUnavailable)
		return
	}

	// Read the body first for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("failed to read response body", "error", err)
		http.Error(w, "Failed to read market data", http.StatusInternalServerError)
		return
	}

	h.logger.Info("response body preview", "first_500_chars", string(bodyBytes[:min(500, len(bodyBytes))]))

	var chainData NasdaqOptionChain
	if err := json.Unmarshal(bodyBytes, &chainData); err != nil {
		h.logger.Error("failed to decode nasdaq response", "error", err, "body_length", len(bodyBytes))
		http.Error(w, fmt.Sprintf("Invalid market data: %v", err), http.StatusInternalServerError)
		return
	}

	h.logger.Info("decoded nasdaq data",
		"total_records", chainData.Data.TotalRecord,
		"rows_count", len(chainData.Data.Table.Rows),
		"status_code", chainData.Status.RCode,
		"status_message", chainData.Status.BCodeMessage,
		"dev_message", chainData.Status.DeveloperMessage)

	// Check if Nasdaq returned an error
	if chainData.Status.RCode != 200 {
		h.logger.Error("nasdaq api error response",
			"code", chainData.Status.RCode,
			"message", chainData.Status.BCodeMessage,
			"dev_message", chainData.Status.DeveloperMessage)
		http.Error(w, fmt.Sprintf("Nasdaq API error: %s", chainData.Status.BCodeMessage), http.StatusServiceUnavailable)
		return
	}

	// Parse and analyze options data
	analysis, err := h.analyzeOptions(chainData, expiryDate, "SPY")
	if err != nil {
		h.logger.Error("failed to analyze options", "error", err)
		http.Error(w, fmt.Sprintf("Analysis failed: %v (rows=%d)", err, len(chainData.Data.Table.Rows)), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analysis)
}

func (h *OptionsAnalysisHandler) HandleIBITRiskAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("fetching IBIT options chain for risk analysis")

	// Use January 15, 2027 for IBIT options (furthest available ~15 months out)
	// Note: IBIT has leap options available for longer-term analysis
	expiryDate := "2027-01-15"

	h.logger.Info("using expiry date", "date", expiryDate)

	// Fetch options data from Nasdaq (limit=200 to get full strike range around current price)
	// Note: IBIT uses assetclass=stocks, not etf
	nasdaqURL := fmt.Sprintf("https://api.nasdaq.com/api/quote/IBIT/option-chain?assetclass=stocks&limit=200&fromdate=%s&todate=%s&excode=oprac&callput=callput&money=all&type=all",
		expiryDate, expiryDate)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", nasdaqURL, nil)
	if err != nil {
		h.logger.Error("failed to create nasdaq request", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set comprehensive headers to mimic real browser
	// Note: Don't set Accept-Encoding manually - let Go handle compression automatically
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.nasdaq.com/")
	req.Header.Set("Origin", "https://www.nasdaq.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	h.logger.Info("requesting nasdaq data", "url", nasdaqURL)

	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error("failed to fetch nasdaq data", "error", err)
		http.Error(w, "Failed to fetch market data", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	h.logger.Info("nasdaq response received", "status", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"))

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("nasdaq api returned non-200", "status", resp.StatusCode)
		http.Error(w, fmt.Sprintf("Market data unavailable (HTTP %d)", resp.StatusCode), http.StatusServiceUnavailable)
		return
	}

	// Read the body first for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("failed to read response body", "error", err)
		http.Error(w, "Failed to read market data", http.StatusInternalServerError)
		return
	}

	h.logger.Info("response body preview", "first_500_chars", string(bodyBytes[:min(500, len(bodyBytes))]))

	var chainData NasdaqOptionChain
	if err := json.Unmarshal(bodyBytes, &chainData); err != nil {
		h.logger.Error("failed to decode nasdaq response", "error", err, "body_length", len(bodyBytes))
		http.Error(w, fmt.Sprintf("Invalid market data: %v", err), http.StatusInternalServerError)
		return
	}

	h.logger.Info("decoded nasdaq data",
		"total_records", chainData.Data.TotalRecord,
		"rows_count", len(chainData.Data.Table.Rows),
		"status_code", chainData.Status.RCode,
		"status_message", chainData.Status.BCodeMessage,
		"dev_message", chainData.Status.DeveloperMessage)

	// Check if Nasdaq returned an error
	if chainData.Status.RCode != 200 {
		h.logger.Error("nasdaq api error response",
			"code", chainData.Status.RCode,
			"message", chainData.Status.BCodeMessage,
			"dev_message", chainData.Status.DeveloperMessage)
		http.Error(w, fmt.Sprintf("Nasdaq API error: %s", chainData.Status.BCodeMessage), http.StatusServiceUnavailable)
		return
	}

	// Parse and analyze options data
	analysis, err := h.analyzeOptions(chainData, expiryDate, "IBIT")
	if err != nil {
		h.logger.Error("failed to analyze options", "error", err)
		http.Error(w, fmt.Sprintf("Analysis failed: %v (rows=%d)", err, len(chainData.Data.Table.Rows)), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analysis)
}

func (h *OptionsAnalysisHandler) HandleGLDRiskAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("fetching GLD options chain for risk analysis")

	// Use September 2026 for GLD options (~1 year out, 335 days)
	// Closest available to 1-year horizon with adequate liquidity
	expiryDate := "2026-09-18"

	h.logger.Info("using expiry date", "date", expiryDate)

	// Fetch options data from Nasdaq (limit=200 to get full strike range around current price)
	// GLD is an ETF, so use assetclass=etf
	nasdaqURL := fmt.Sprintf("https://api.nasdaq.com/api/quote/GLD/option-chain?assetclass=etf&limit=200&fromdate=%s&todate=%s&excode=oprac&callput=callput&money=all&type=all",
		expiryDate, expiryDate)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", nasdaqURL, nil)
	if err != nil {
		h.logger.Error("failed to create nasdaq request", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set comprehensive headers to mimic real browser
	// Note: Don't set Accept-Encoding manually - let Go handle compression automatically
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.nasdaq.com/")
	req.Header.Set("Origin", "https://www.nasdaq.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	h.logger.Info("requesting nasdaq data", "url", nasdaqURL)

	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error("failed to fetch nasdaq data", "error", err)
		http.Error(w, "Failed to fetch market data", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	h.logger.Info("nasdaq response received", "status", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"))

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("nasdaq api returned non-200", "status", resp.StatusCode)
		http.Error(w, fmt.Sprintf("Market data unavailable (HTTP %d)", resp.StatusCode), http.StatusServiceUnavailable)
		return
	}

	// Read the body first for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("failed to read response body", "error", err)
		http.Error(w, "Failed to read market data", http.StatusInternalServerError)
		return
	}

	h.logger.Info("response body preview", "first_500_chars", string(bodyBytes[:min(500, len(bodyBytes))]))

	var chainData NasdaqOptionChain
	if err := json.Unmarshal(bodyBytes, &chainData); err != nil {
		h.logger.Error("failed to decode nasdaq response", "error", err, "body_length", len(bodyBytes))
		http.Error(w, fmt.Sprintf("Invalid market data: %v", err), http.StatusInternalServerError)
		return
	}

	h.logger.Info("decoded nasdaq data",
		"total_records", chainData.Data.TotalRecord,
		"rows_count", len(chainData.Data.Table.Rows),
		"status_code", chainData.Status.RCode,
		"status_message", chainData.Status.BCodeMessage,
		"dev_message", chainData.Status.DeveloperMessage)

	// Check if Nasdaq returned an error
	if chainData.Status.RCode != 200 {
		h.logger.Error("nasdaq api error response",
			"code", chainData.Status.RCode,
			"message", chainData.Status.BCodeMessage,
			"dev_message", chainData.Status.DeveloperMessage)
		http.Error(w, fmt.Sprintf("Nasdaq API error: %s", chainData.Status.BCodeMessage), http.StatusServiceUnavailable)
		return
	}

	// Parse and analyze options data
	analysis, err := h.analyzeOptions(chainData, expiryDate, "GLD")
	if err != nil {
		h.logger.Error("failed to analyze options", "error", err)
		http.Error(w, fmt.Sprintf("Analysis failed: %v (rows=%d)", err, len(chainData.Data.Table.Rows)), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analysis)
}

func (h *OptionsAnalysisHandler) analyzeOptions(chainData NasdaqOptionChain, expiryDate string, symbol string) (*RiskAnalysisResponse, error) {
	// Calculate days to expiry first (needed for IV calculations)
	expiryTime, _ := time.Parse("2006-01-02", expiryDate)
	daysToExpiry := int(time.Until(expiryTime).Hours() / 24)

	// Parse options data into structured format (V2 with better filtering)
	options, currentPrice, err := h.parseOptionsDataV2(chainData, daysToExpiry)
	if err != nil {
		return nil, err
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no valid options data found")
	}

	analysis := &RiskAnalysisResponse{
		Timestamp:    time.Now().Format(time.RFC3339),
		Symbol:       symbol,
		CurrentPrice: currentPrice,
		DaysToExpiry: daysToExpiry,
		DataQuality: DataQuality{
			OptionsAnalyzed: len(options),
			Warnings:        []string{},
		},
	}

	// Calculate risk-neutral probabilities using improved Breeden-Litzenberger
	analysis.RiskNeutralProbabilities = h.calculateRiskNeutralProbsV2(options, currentPrice)

	// Calculate implied volatility metrics (with Black-Scholes IV)
	analysis.ImpliedVolatilityMetrics = h.calculateIVMetricsV2(options, currentPrice)

	// Calculate market-implied expected return
	analysis.MarketExpectedReturn = h.calculateExpectedReturnV2(options, currentPrice, daysToExpiry)

	// Calculate tail risk metrics
	analysis.TailRiskMetrics = h.calculateTailRisk(options, currentPrice)

	// Calculate skew metrics
	analysis.SkewMetrics = h.calculateSkewMetrics(options, currentPrice)

	// Calculate put/call ratio (using open interest for better accuracy)
	analysis.PutCallRatio = h.calculatePutCallRatioV2(options)

	// Add data quality metrics
	h.addDataQualityMetrics(analysis, options)

	return analysis, nil
}

func (h *OptionsAnalysisHandler) calculateTailRisk(options []OptionData, currentPrice float64) TailRisk {
	risk := TailRisk{}

	// Find 5th and 95th percentile strikes
	if len(options) > 10 {
		idx5 := len(options) / 20       // 5th percentile
		idx95 := len(options) * 19 / 20 // 95th percentile

		if idx5 < len(options) && idx95 < len(options) {
			leftTail := options[idx5].Strike
			rightTail := options[idx95].Strike

			risk.LeftTailRisk = (leftTail - currentPrice) / currentPrice * 100
			risk.RightTailRisk = (rightTail - currentPrice) / currentPrice * 100
		}
	}

	// Expected shortfall (average of worst 5% outcomes)
	cutoff := len(options) / 20
	if cutoff > 0 && cutoff < len(options) {
		var sum float64
		for i := 0; i < cutoff; i++ {
			sum += options[i].Strike
		}
		avgWorstCase := sum / float64(cutoff)
		risk.ExpectedShortfall = (avgWorstCase - currentPrice) / currentPrice * 100
	}

	// Kurtosis proxy (ratio of tails)
	if risk.RightTailRisk != 0 && risk.LeftTailRisk != 0 {
		risk.KurtosisProxy = math.Abs(risk.LeftTailRisk) / risk.RightTailRisk
	}

	return risk
}

func (h *OptionsAnalysisHandler) calculateSkewMetrics(options []OptionData, currentPrice float64) SkewMetrics {
	metrics := SkewMetrics{}

	// Risk reversal: IV of OTM put - IV of OTM call
	var otmPut, otmCall OptionData
	for _, opt := range options {
		if opt.Strike < currentPrice*0.9 && opt.PutIV > 0 && otmPut.Strike == 0 {
			otmPut = opt
		}
		if opt.Strike > currentPrice*1.1 && opt.CallIV > 0 && otmCall.Strike == 0 {
			otmCall = opt
		}
	}

	if otmPut.PutIV > 0 && otmCall.CallIV > 0 {
		metrics.RiskReversalSkew = (otmPut.PutIV - otmCall.CallIV) * 100
	}

	// Butterfly spread
	if len(options) > 3 {
		mid := len(options) / 2
		if mid > 0 && mid < len(options)-1 {
			c1 := options[mid-1].CallMid
			c2 := options[mid].CallMid
			c3 := options[mid+1].CallMid

			if c1 > 0 && c2 > 0 && c3 > 0 {
				metrics.ButterflySpread = c1 - 2*c2 + c3
			}
		}
	}

	// Skewness estimate from option prices
	metrics.SkewnessEstimate = metrics.RiskReversalSkew / 100

	return metrics
}

func (h *OptionsAnalysisHandler) addDataQualityMetrics(analysis *RiskAnalysisResponse, options []OptionData) {
	if len(options) == 0 {
		return
	}

	// Strike range
	minStrike := options[0].Strike
	maxStrike := options[len(options)-1].Strike
	analysis.DataQuality.StrikeRange = fmt.Sprintf("$%.2f - $%.2f", minStrike, maxStrike)

	// Average bid-ask spread
	var totalSpread float64
	var count int

	for _, opt := range options {
		if opt.CallAsk > 0 && opt.CallBid > 0 {
			spread := (opt.CallAsk - opt.CallBid) / opt.CallMid * 100
			totalSpread += spread
			count++
		}
	}

	if count > 0 {
		analysis.DataQuality.AvgBidAskSpread = totalSpread / float64(count)
	}

	// Add warnings
	if analysis.DataQuality.OptionsAnalyzed < 20 {
		analysis.DataQuality.Warnings = append(analysis.DataQuality.Warnings,
			"Limited options data available - results may be less reliable")
	}

	if analysis.DataQuality.AvgBidAskSpread > 5 {
		analysis.DataQuality.Warnings = append(analysis.DataQuality.Warnings,
			"Wide bid-ask spreads detected - liquidity may be low")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseFloat(s string) float64 {
	// Remove $ and % symbols, parse as float
	s = strings.Replace(s, "$", "", -1)
	s = strings.Replace(s, "%", "", -1)
	s = strings.TrimSpace(s)

	if s == "" || s == "--" || s == "N/A" {
		return 0
	}

	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}

func (h *OptionsAnalysisHandler) HandleTLTRiskAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("fetching TLT options chain for risk analysis")
	expiryDate := "2026-09-18"
	h.logger.Info("using expiry date", "date", expiryDate)

	nasdaqURL := fmt.Sprintf("https://api.nasdaq.com/api/quote/TLT/option-chain?assetclass=etf&limit=200&fromdate=%s&todate=%s&excode=oprac&callput=callput&money=all&type=all",
		expiryDate, expiryDate)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", nasdaqURL, nil)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.nasdaq.com/")
	req.Header.Set("Origin", "https://www.nasdaq.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch market data", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Market data unavailable (HTTP %d)", resp.StatusCode), http.StatusServiceUnavailable)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read market data", http.StatusInternalServerError)
		return
	}

	var chainData NasdaqOptionChain
	if err := json.Unmarshal(bodyBytes, &chainData); err != nil {
		http.Error(w, fmt.Sprintf("Invalid market data: %v", err), http.StatusInternalServerError)
		return
	}

	if chainData.Status.RCode != 200 {
		http.Error(w, fmt.Sprintf("Nasdaq API error: %s", chainData.Status.BCodeMessage), http.StatusServiceUnavailable)
		return
	}

	analysis, err := h.analyzeOptions(chainData, expiryDate, "TLT")
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analysis)
}

func (h *OptionsAnalysisHandler) HandleVNQRiskAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("fetching VNQ options chain for risk analysis")

	// Use November 21, 2025 for VNQ options (only available expiry as of Oct 2025)
	expiryDate := "2025-11-21"
	h.logger.Info("using expiry date", "date", expiryDate)

	// VNQ is an ETF, so use assetclass=etf
	nasdaqURL := fmt.Sprintf("https://api.nasdaq.com/api/quote/VNQ/option-chain?assetclass=etf&limit=200&fromdate=%s&todate=%s&excode=oprac&callput=callput&money=all&type=all",
		expiryDate, expiryDate)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", nasdaqURL, nil)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.nasdaq.com/")
	req.Header.Set("Origin", "https://www.nasdaq.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch market data", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Market data unavailable (HTTP %d)", resp.StatusCode), http.StatusServiceUnavailable)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read market data", http.StatusInternalServerError)
		return
	}

	var chainData NasdaqOptionChain
	if err := json.Unmarshal(bodyBytes, &chainData); err != nil {
		http.Error(w, fmt.Sprintf("Invalid market data: %v", err), http.StatusInternalServerError)
		return
	}

	if chainData.Status.RCode != 200 {
		http.Error(w, fmt.Sprintf("Nasdaq API error: %s", chainData.Status.BCodeMessage), http.StatusServiceUnavailable)
		return
	}

	analysis, err := h.analyzeOptions(chainData, expiryDate, "VNQ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analysis)
}

func (h *OptionsAnalysisHandler) HandleUSORiskAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("fetching USO options chain for risk analysis")

	// Use January 15, 2027 for USO options (LEAPS, ~15 months out - closest available to 1 year)
	// Note: USO has limited option expirations, this is the furthest available
	expiryDate := "2027-01-15"
	h.logger.Info("using expiry date", "date", expiryDate)

	// USO is an ETF, so use assetclass=etf
	nasdaqURL := fmt.Sprintf("https://api.nasdaq.com/api/quote/USO/option-chain?assetclass=etf&limit=200&fromdate=%s&todate=%s&excode=oprac&callput=callput&money=all&type=all",
		expiryDate, expiryDate)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", nasdaqURL, nil)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.nasdaq.com/")
	req.Header.Set("Origin", "https://www.nasdaq.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch market data", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Market data unavailable (HTTP %d)", resp.StatusCode), http.StatusServiceUnavailable)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read market data", http.StatusInternalServerError)
		return
	}

	var chainData NasdaqOptionChain
	if err := json.Unmarshal(bodyBytes, &chainData); err != nil {
		http.Error(w, fmt.Sprintf("Invalid market data: %v", err), http.StatusInternalServerError)
		return
	}

	if chainData.Status.RCode != 200 {
		http.Error(w, fmt.Sprintf("Nasdaq API error: %s", chainData.Status.BCodeMessage), http.StatusServiceUnavailable)
		return
	}

	analysis, err := h.analyzeOptions(chainData, expiryDate, "USO")
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analysis)
}
