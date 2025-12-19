import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { Header } from '../components/Header';
import { API_BASE_URL } from '../utils/api';
import { TrendingDown, TrendingUp, Activity, AlertTriangle } from 'lucide-react';

interface RiskNeutralProbs {
  prob_gain_0pct_plus: number;
  prob_gain_5pct_plus: number;
  prob_gain_10pct_plus: number;
  prob_gain_15pct_plus: number;
  prob_loss_0pct_plus: number;
  prob_loss_5pct_plus: number;
  prob_loss_10pct_plus: number;
  prob_loss_15pct_plus: number;
}

interface IVMetrics {
  atm_implied_vol_percent: number;
  iv_skew: number;
  iv_term_structure: string;
  vix_equivalent_percent: number;
}

interface TailRisk {
  left_tail_risk_5pct: number;
  right_tail_risk_95pct: number;
  expected_shortfall_percent: number;
  kurtosis_proxy: number;
}

interface SkewMetrics {
  risk_reversal_25delta: number;
  butterfly_spread: number;
  skewness_estimate: number;
}

interface DataQuality {
  options_analyzed: number;
  strike_range: string;
  avg_bid_ask_spread_percent: number;
  warnings: string[];
}

interface RiskAnalysis {
  timestamp: string;
  symbol: string;
  current_price: number;
  days_to_expiry: number;
  risk_neutral_probabilities: RiskNeutralProbs;
  implied_volatility_metrics: IVMetrics;
  market_expected_return_percent: number;
  tail_risk_metrics: TailRisk;
  skew_metrics: SkewMetrics;
  put_call_ratio: number;
  data_quality: DataQuality;
}

const TICKER_INFO: Record<string, { name: string; description: string }> = {
  SPY: { name: 'S&P 500 ETF', description: 'SPDR S&P 500 ETF Trust' },
  IBIT: { name: 'Bitcoin ETF', description: 'iShares Bitcoin Trust' },
  GLD: { name: 'Gold ETF', description: 'SPDR Gold Shares' },
  TLT: { name: '20+ Year Treasury ETF', description: 'iShares 20+ Year Treasury Bond ETF' },
  VNQ: { name: 'Real Estate ETF', description: 'Vanguard Real Estate ETF' },
  USO: { name: 'Oil ETF', description: 'United States Oil Fund' },
};

export function RiskAnalysisPage() {
  const { ticker } = useParams<{ ticker: string }>();
  const [analysis, setAnalysis] = useState<RiskAnalysis | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const tickerUpper = ticker?.toUpperCase() || '';
  const tickerInfo = TICKER_INFO[tickerUpper] || { name: tickerUpper, description: tickerUpper };

  useEffect(() => {
    const fetchAnalysis = async () => {
      if (!ticker) return;

      setLoading(true);
      setError(null);

      try {
        const cacheBuster = `?_=${Date.now()}`;
        const response = await fetch(`${API_BASE_URL}/api/market/${ticker.toLowerCase()}-risk-analysis${cacheBuster}`);
        if (!response.ok) {
          throw new Error(`Failed to fetch analysis (HTTP ${response.status})`);
        }
        const data = await response.json();
        setAnalysis(data);
      } catch (err) {
        console.error('Error fetching risk analysis:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchAnalysis();
  }, [ticker]);

  const formatPercent = (value: number, decimals: number = 1) => {
    return `${(value).toFixed(decimals)}%`;
  };

  return (
    <div className="min-h-screen bg-void text-chalk">
      <div className="scan-line" />
      <Header />

      <div className="pt-16">
        <main className="max-w-7xl mx-auto p-4 md:p-8 space-y-6">
          {/* Page Header */}
          <div className="border-l-4 border-terminal pl-4 md:pl-6">
            <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-2">
              <div>
                <h1 className="font-display font-black text-3xl md:text-4xl text-chalk tracking-tight">
                  {tickerUpper} RISK ANALYSIS
                </h1>
                <p className="text-sm md:text-base text-smoke font-mono mt-2">
                  {tickerInfo.name} / {tickerInfo.description}
                </p>
              </div>
              {analysis && (
                <div className="text-xs font-mono text-fog">
                  DATA AS OF: {new Date(analysis.timestamp).toLocaleString('en-US', {
                    dateStyle: 'short',
                    timeStyle: 'short'
                  })}
                </div>
              )}
            </div>
          </div>

          {/* Loading State */}
          {loading && (
            <div className="border-2 border-steel bg-concrete p-16 text-center space-y-6">
              <div className="text-5xl text-terminal animate-pulse">◉</div>
              <div className="space-y-2">
                <p className="text-chalk font-mono text-lg font-bold">LOADING ANALYSIS...</p>
                <p className="text-fog font-mono text-sm">Fetching options data from market</p>
              </div>
            </div>
          )}

          {/* Error State */}
          {error && (
            <div className="border-2 border-warning bg-concrete p-16 text-center space-y-6">
              <div className="text-5xl text-warning">⚠</div>
              <div className="space-y-2">
                <p className="text-chalk font-mono text-lg font-bold">ANALYSIS ERROR</p>
                <p className="text-fog font-mono text-sm">{error}</p>
              </div>
            </div>
          )}

          {/* Analysis Content */}
          {analysis && !loading && (
            <>
              {/* Current Price & Summary */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div className="border-2 border-terminal bg-concrete p-6">
                  <div className="text-xs font-mono text-smoke mb-2">CURRENT PRICE</div>
                  <div className="text-3xl font-display font-black text-terminal">
                    ${analysis.current_price.toFixed(2)}
                  </div>
                </div>

                <div className="border-2 border-steel bg-concrete p-6">
                  <div className="text-xs font-mono text-smoke mb-2">DAYS TO EXPIRY</div>
                  <div className="text-3xl font-display font-black text-chalk">
                    {analysis.days_to_expiry}
                  </div>
                </div>

                <div className="border-2 border-steel bg-concrete p-6">
                  <div className="text-xs font-mono text-smoke mb-2">IMPLIED VOLATILITY</div>
                  <div className="text-3xl font-display font-black text-electric">
                    {formatPercent(analysis.implied_volatility_metrics.atm_implied_vol_percent)}
                  </div>
                </div>

                <div className="border-2 border-steel bg-concrete p-6">
                  <div className="text-xs font-mono text-smoke mb-2">PUT/CALL RATIO</div>
                  <div className="text-3xl font-display font-black text-chalk">
                    {analysis.put_call_ratio.toFixed(2)}
                  </div>
                </div>
              </div>

              {/* Risk-Neutral Probabilities */}
              <div className="border-2 border-terminal bg-concrete p-6">
                <h2 className="font-mono font-bold text-lg text-terminal mb-4 flex items-center gap-2">
                  <Activity className="w-5 h-5" />
                  RISK-NEUTRAL PROBABILITIES
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <h3 className="text-sm font-mono text-smoke mb-3 flex items-center gap-2">
                      <TrendingUp className="w-4 h-4 text-terminal" />
                      UPSIDE PROBABILITIES
                    </h3>
                    <div className="space-y-2 text-sm font-mono">
                      <div className="flex justify-between">
                        <span className="text-smoke">Gain 0%+</span>
                        <span className="text-terminal font-bold">{formatPercent(analysis.risk_neutral_probabilities.prob_gain_0pct_plus * 100)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-smoke">Gain 5%+</span>
                        <span className="text-terminal font-bold">{formatPercent(analysis.risk_neutral_probabilities.prob_gain_5pct_plus * 100)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-smoke">Gain 10%+</span>
                        <span className="text-terminal font-bold">{formatPercent(analysis.risk_neutral_probabilities.prob_gain_10pct_plus * 100)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-smoke">Gain 15%+</span>
                        <span className="text-terminal font-bold">{formatPercent(analysis.risk_neutral_probabilities.prob_gain_15pct_plus * 100)}</span>
                      </div>
                    </div>
                  </div>

                  <div>
                    <h3 className="text-sm font-mono text-smoke mb-3 flex items-center gap-2">
                      <TrendingDown className="w-4 h-4 text-warning" />
                      DOWNSIDE PROBABILITIES
                    </h3>
                    <div className="space-y-2 text-sm font-mono">
                      <div className="flex justify-between">
                        <span className="text-smoke">Loss 0%+</span>
                        <span className="text-warning font-bold">{formatPercent(analysis.risk_neutral_probabilities.prob_loss_0pct_plus * 100)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-smoke">Loss 5%+</span>
                        <span className="text-warning font-bold">{formatPercent(analysis.risk_neutral_probabilities.prob_loss_5pct_plus * 100)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-smoke">Loss 10%+</span>
                        <span className="text-warning font-bold">{formatPercent(analysis.risk_neutral_probabilities.prob_loss_10pct_plus * 100)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-smoke">Loss 15%+</span>
                        <span className="text-warning font-bold">{formatPercent(analysis.risk_neutral_probabilities.prob_loss_15pct_plus * 100)}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Tail Risk & Skew */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="border-2 border-steel bg-concrete p-6">
                  <h2 className="font-mono font-bold text-lg text-chalk mb-4">TAIL RISK METRICS</h2>
                  <div className="space-y-2 text-sm font-mono">
                    <div className="flex justify-between">
                      <span className="text-smoke">Left Tail (5%)</span>
                      <span className="text-warning font-bold">{formatPercent(analysis.tail_risk_metrics.left_tail_risk_5pct)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-smoke">Right Tail (95%)</span>
                      <span className="text-terminal font-bold">{formatPercent(analysis.tail_risk_metrics.right_tail_risk_95pct)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-smoke">Expected Shortfall</span>
                      <span className="text-warning font-bold">{formatPercent(analysis.tail_risk_metrics.expected_shortfall_percent)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-smoke">Kurtosis Proxy</span>
                      <span className="text-chalk font-bold">{analysis.tail_risk_metrics.kurtosis_proxy.toFixed(2)}</span>
                    </div>
                  </div>
                </div>

                <div className="border-2 border-steel bg-concrete p-6">
                  <h2 className="font-mono font-bold text-lg text-chalk mb-4">SKEW METRICS</h2>
                  <div className="space-y-2 text-sm font-mono">
                    <div className="flex justify-between">
                      <span className="text-smoke">Risk Reversal (25Δ)</span>
                      <span className="text-chalk font-bold">{formatPercent(analysis.skew_metrics.risk_reversal_25delta)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-smoke">Butterfly Spread</span>
                      <span className="text-chalk font-bold">{analysis.skew_metrics.butterfly_spread.toFixed(4)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-smoke">Skewness Estimate</span>
                      <span className="text-chalk font-bold">{analysis.skew_metrics.skewness_estimate.toFixed(3)}</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Volatility Metrics */}
              <div className="border-2 border-electric bg-concrete p-6">
                <h2 className="font-mono font-bold text-lg text-electric mb-4">IMPLIED VOLATILITY METRICS</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm font-mono">
                  <div className="flex justify-between">
                    <span className="text-smoke">ATM Implied Vol</span>
                    <span className="text-electric font-bold">{formatPercent(analysis.implied_volatility_metrics.atm_implied_vol_percent)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-smoke">VIX Equivalent</span>
                    <span className="text-electric font-bold">{formatPercent(analysis.implied_volatility_metrics.vix_equivalent_percent)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-smoke">IV Skew</span>
                    <span className="text-electric font-bold">{formatPercent(analysis.implied_volatility_metrics.iv_skew)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-smoke">Term Structure</span>
                    <span className="text-electric font-bold">{analysis.implied_volatility_metrics.iv_term_structure}</span>
                  </div>
                </div>
              </div>

              {/* Expected Return */}
              <div className="border-2 border-steel bg-concrete p-6">
                <h2 className="font-mono font-bold text-lg text-chalk mb-2">MARKET EXPECTED RETURN</h2>
                <div className="text-4xl font-display font-black text-terminal">
                  {formatPercent(analysis.market_expected_return_percent, 2)}
                </div>
                <p className="text-xs font-mono text-smoke mt-2">Annual risk-neutral expected return</p>
              </div>

              {/* Data Quality */}
              <div className="border-2 border-steel bg-concrete p-6">
                <h2 className="font-mono font-bold text-lg text-chalk mb-4 flex items-center gap-2">
                  <AlertTriangle className="w-5 h-5" />
                  DATA QUALITY
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm font-mono mb-4">
                  <div className="flex justify-between">
                    <span className="text-smoke">Options Analyzed</span>
                    <span className="text-chalk font-bold">{analysis.data_quality.options_analyzed}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-smoke">Strike Range</span>
                    <span className="text-chalk font-bold">{analysis.data_quality.strike_range}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-smoke">Avg Bid-Ask Spread</span>
                    <span className="text-chalk font-bold">{formatPercent(analysis.data_quality.avg_bid_ask_spread_percent)}</span>
                  </div>
                </div>

                {analysis.data_quality.warnings.length > 0 && (
                  <div className="mt-4 space-y-2">
                    {analysis.data_quality.warnings.map((warning, index) => (
                      <div key={index} className="bg-warning/10 border-2 border-warning p-3 text-xs font-mono text-warning">
                        ⚠ {warning}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Methodology Note */}
              <div className="border-2 border-steel bg-concrete p-6">
                <h2 className="font-mono font-bold text-sm text-smoke mb-2">METHODOLOGY</h2>
                <p className="text-xs font-mono text-fog leading-relaxed">
                  Risk-neutral probabilities derived from options prices using the Breeden-Litzenberger formula.
                  Implied volatilities calculated using Black-Scholes model. Tail risk metrics computed from
                  5th and 95th percentile strikes. All metrics are forward-looking and reflect market
                  expectations embedded in options prices. Past performance does not guarantee future results.
                </p>
              </div>
            </>
          )}
        </main>
      </div>

      {/* Footer */}
      <footer className="border-t border-steel bg-concrete mt-12">
        <div className="px-4 md:px-6 py-4 flex flex-col md:flex-row items-center justify-between gap-3 text-xs font-mono text-smoke">
          <div className="text-center md:text-left">STRATINT v1.0</div>
          <div className="flex items-center gap-4">
            <a href="/" className="hover:text-terminal transition-colors">HOME</a>
            <span>|</span>
            <a href="/api-docs" className="hover:text-terminal transition-colors">API DOCS</a>
          </div>
        </div>
      </footer>
    </div>
  );
}
