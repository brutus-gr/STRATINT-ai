import { useState, useEffect, useRef } from 'react';
import {
  Line,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ComposedChart,
} from 'recharts';
import { createChart, ColorType } from 'lightweight-charts';
import { Loader } from 'lucide-react';
import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';

interface ChartDataPoint {
  timestamp: string;
  date: Date;
  p10: number;
  p25: number;
  p50: number;
  p75: number;
  p90: number;
  // For area charts, we need the difference from median
  p25_p75_lower: number;
  p25_p75_upper: number;
  p10_p90_lower: number;
  p10_p90_upper: number;
}

interface OHLCDataPoint {
  date: string;
  open: number;
  high: number;
  low: number;
  close: number;
}

interface ForecastChartProps {
  forecastId: string;
  viewMode: 'hourly' | '4h' | 'daily';
}

// Lightweight Charts OHLC Component
function LightweightOHLCChart({ data }: { data: OHLCDataPoint[] }) {
  const chartContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!chartContainerRef.current || !data || data.length === 0) return;

    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: '#1a1a1a' },
        textColor: '#d3d3d3',
      },
      grid: {
        vertLines: { color: '#3a3a3a' },
        horzLines: { color: '#3a3a3a' },
      },
      width: chartContainerRef.current.clientWidth,
      height: 400,
    });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const candlestickSeries = (chart as any).addCandlestickSeries({
      upColor: '#00ff95',
      downColor: '#ff4444',
      borderUpColor: '#00ff95',
      borderDownColor: '#ff4444',
      wickUpColor: '#808080',
      wickDownColor: '#808080',
    });

    // Convert data to Lightweight Charts format
    // Daily data: time is 'YYYY-MM-DD' string
    // 4h data: time is Unix timestamp (parse as number)
    const formattedData = data.map(item => {
      const timeValue = /^\d+$/.test(item.date) ? parseInt(item.date, 10) as any : item.date;
      return {
        time: timeValue,
        open: item.open,
        high: item.high,
        low: item.low,
        close: item.close,
      };
    });

    candlestickSeries.setData(formattedData);
    chart.timeScale().fitContent();

    // Handle resize
    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({ width: chartContainerRef.current.clientWidth });
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      chart.remove();
    };
  }, [data]);

  return <div ref={chartContainerRef} style={{ width: '100%', height: '400px' }} />;
}

export function ForecastChart({ forecastId, viewMode }: ForecastChartProps) {
  const [loading, setLoading] = useState(true);
  const [hourlyData, setHourlyData] = useState<ChartDataPoint[]>([]);
  const [fourHourData, setFourHourData] = useState<OHLCDataPoint[]>([]);
  const [dailyData, setDailyData] = useState<OHLCDataPoint[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchHistory();
  }, [forecastId]);

  const fetchHistory = async () => {
    try {
      setLoading(true);
      setError(null);

      // Fetch hourly data
      const hourlyResponse = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecastId}/history`, {
        headers: getAuthHeaders(),
      });

      if (!hourlyResponse.ok) {
        throw new Error('Failed to fetch forecast history');
      }

      const hourlyResult = await hourlyResponse.json();
      const history = hourlyResult.history || [];

      // Transform hourly data for charting - limit to most recent 24 points
      const allHourlyData: ChartDataPoint[] = history
        .filter((item: any) => item.result?.aggregated_percentiles)
        .map((item: any) => {
          const p = item.result.aggregated_percentiles;
          const runDate = new Date(item.run.run_at);
          const timestamp = runDate.toLocaleString('en-US', {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
          });

          return {
            timestamp,
            date: runDate,
            p10: p.p10,
            p25: p.p25,
            p50: p.p50,
            p75: p.p75,
            p90: p.p90,
            p25_p75_lower: p.p25,
            p25_p75_upper: p.p75,
            p10_p90_lower: p.p10,
            p10_p90_upper: p.p90,
          };
        });

      // Take only the last 24 points
      const hourlyChartData = allHourlyData.slice(-24);

      setHourlyData(hourlyChartData);

      // Fetch 4-hour OHLC data
      const fourHourResponse = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecastId}/history/4h`, {
        headers: getAuthHeaders(),
      });

      if (fourHourResponse.ok) {
        const fourHourResult = await fourHourResponse.json();
        setFourHourData(fourHourResult.data || []);
      }

      // Fetch daily OHLC data
      const dailyResponse = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecastId}/history/daily`, {
        headers: getAuthHeaders(),
      });

      if (dailyResponse.ok) {
        const dailyResult = await dailyResponse.json();
        const ohlcData = dailyResult.data || [];
        setDailyData(ohlcData);
      } else {
        console.error('Admin: Failed to fetch daily data', dailyResponse.status);
      }

      setLoading(false);
    } catch (err) {
      console.error('Error fetching forecast history:', err);
      setError(err instanceof Error ? err.message : 'Failed to load history');
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="border-2 border-steel bg-concrete p-8 text-center">
        <Loader className="w-8 h-8 text-terminal/50 mx-auto mb-2 animate-spin" />
        <p className="text-sm font-mono text-fog">Loading forecast history...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="border-2 border-threat-critical bg-threat-critical/10 p-4 text-center">
        <p className="text-sm font-mono text-threat-critical">{error}</p>
      </div>
    );
  }

  const currentData = viewMode === 'hourly' ? hourlyData : (viewMode === '4h' ? fourHourData : dailyData);

  if (currentData.length === 0) {
    return (
      <div className="border-2 border-steel bg-concrete p-8 text-center">
        <p className="text-sm font-mono text-fog">
          No {viewMode === 'hourly' ? 'hourly' : 'daily'} data available. Run the forecast a few times to see trends.
        </p>
      </div>
    );
  }

  // Custom tooltip for hourly view
  const HourlyTooltip = ({ active, payload, label, coordinate }: any) => {
    if (!active || !payload || payload.length === 0) return null;

    const data = payload[0]?.payload;
    if (!data) return null;

    const key = coordinate ? `${coordinate.x}-${coordinate.y}` : 'tooltip';

    return (
      <div key={key} className="border-2 border-terminal bg-void p-3 font-mono text-xs">
        <p className="text-chalk font-bold mb-2">{label}</p>
        <div className="space-y-1 text-smoke">
          <p>P90: <span className="text-chalk font-bold">{data.p90?.toFixed(2)}%</span></p>
          <p>P75: <span className="text-chalk font-bold">{data.p75?.toFixed(2)}%</span></p>
          <p className="text-terminal font-bold">P50: <span className="text-terminal font-bold">{data.p50?.toFixed(2)}%</span></p>
          <p>P25: <span className="text-chalk font-bold">{data.p25?.toFixed(2)}%</span></p>
          <p>P10: <span className="text-chalk font-bold">{data.p10?.toFixed(2)}%</span></p>
        </div>
      </div>
    );
  };

  // DailyTooltip is handled by Lightweight Charts built-in tooltip

  return (
    <div className="border-2 border-steel bg-concrete p-4">
      <div className="flex justify-between items-center mb-4">
        <h4 className="text-sm font-mono text-chalk font-bold">FORECAST HISTORY</h4>
      </div>

      {viewMode === 'hourly' ? (
        <ResponsiveContainer width="100%" height={400}>
          <ComposedChart
            data={hourlyData}
            margin={{ top: 10, right: 30, left: 0, bottom: 0 }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#3a3a3a" />
            <XAxis
              dataKey="timestamp"
              stroke="#808080"
              style={{
                fontSize: '11px',
                fontFamily: 'monospace',
              }}
            />
            <YAxis
              stroke="#808080"
              style={{
                fontSize: '11px',
                fontFamily: 'monospace',
              }}
              tickFormatter={(value) => `${value}%`}
              label={{
                value: 'Probability (%)',
                angle: -90,
                position: 'insideLeft',
                style: { fontSize: '11px', fontFamily: 'monospace', fill: '#808080' },
              }}
            />
            <Tooltip
              content={<HourlyTooltip />}
              cursor={{ stroke: '#00ff95', strokeWidth: 1, strokeDasharray: '5 5' }}
              isAnimationActive={false}
              allowEscapeViewBox={{ x: true, y: true }}
            />
            <Legend
              wrapperStyle={{
                fontSize: '11px',
                fontFamily: 'monospace',
              }}
            />

            {/* P10-P90 confidence band (lightest) */}
            <Area
              type="monotone"
              dataKey="p10_p90_upper"
              stroke="none"
              fill="#00ff9520"
              name="P10-P90 Band"
              isAnimationActive={false}
              activeDot={false}
              tooltipType="none"
            />
            <Area
              type="monotone"
              dataKey="p10_p90_lower"
              stroke="none"
              fill="#ffffff"
              name=""
              isAnimationActive={false}
              activeDot={false}
              tooltipType="none"
            />

            {/* P25-P75 confidence band (darker) */}
            <Area
              type="monotone"
              dataKey="p25_p75_upper"
              stroke="none"
              fill="#00ff9540"
              name="P25-P75 Band"
              isAnimationActive={false}
              activeDot={false}
              tooltipType="none"
            />
            <Area
              type="monotone"
              dataKey="p25_p75_lower"
              stroke="none"
              fill="#ffffff"
              name=""
              isAnimationActive={false}
              activeDot={false}
              tooltipType="none"
            />

            {/* P50 median line (main prediction) */}
            <Line
              type="monotone"
              dataKey="p50"
              stroke="#00ff95"
              strokeWidth={3}
              dot={{ fill: '#00ff95', r: 4 }}
              activeDot={{ r: 6, fill: '#00ff95' }}
              name="Median (P50)"
              isAnimationActive={false}
            />
          </ComposedChart>
        </ResponsiveContainer>
      ) : (
        <LightweightOHLCChart data={viewMode === '4h' ? fourHourData : dailyData} />
      )}

      <div className="mt-4 text-xs font-mono text-fog">
        {viewMode === 'hourly' ? (
          <>
            <p>
              <span className="text-terminal font-bold">■</span> P50 (Median): Main forecast prediction
            </p>
            <p className="mt-1">
              <span className="inline-block w-3 h-3 bg-terminal/25 border border-terminal mr-1"></span>
              P25-P75 Band: 50% confidence interval
            </p>
            <p className="mt-1">
              <span className="inline-block w-3 h-3 bg-terminal/10 border border-terminal mr-1"></span>
              P10-P90 Band: 80% confidence interval
            </p>
          </>
        ) : viewMode === '4h' ? (
          <>
            <p>
              <span className="text-terminal font-bold">■</span> OHLC Bars: 4-hour P50 median prediction range
            </p>
            <p className="mt-1 text-smoke">
              <span className="text-terminal">▲ Green:</span> Close &gt; Open | <span className="text-threat-critical">▼ Red:</span> Close &lt; Open
            </p>
            <p className="mt-1 text-smoke">
              Open: First prediction of 4h period | Close: Last prediction of period
            </p>
          </>
        ) : (
          <>
            <p>
              <span className="text-terminal font-bold">■</span> OHLC Bars: Daily P50 median prediction range
            </p>
            <p className="mt-1 text-smoke">
              <span className="text-terminal">▲ Green:</span> Close &gt; Open | <span className="text-threat-critical">▼ Red:</span> Close &lt; Open
            </p>
            <p className="mt-1 text-smoke">
              Open: First prediction of day | Close: Last prediction of day
            </p>
          </>
        )}
      </div>
    </div>
  );
}
