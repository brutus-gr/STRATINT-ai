package test

import (
	"fmt"
	"html/template"
	"os"
	"time"
)

// TestSuite represents a collection of test results
type TestSuite struct {
	Name        string       `json:"name"`
	StartTime   time.Time    `json:"start_time"`
	EndTime     time.Time    `json:"end_time"`
	TotalTests  int          `json:"total_tests"`
	PassedTests int          `json:"passed_tests"`
	FailedTests int          `json:"failed_tests"`
	Results     []TestResult `json:"results"`
}

// TestResult represents a single test result
type TestResult struct {
	TestName        string                 `json:"test_name"`
	Category        string                 `json:"category"`
	Description     string                 `json:"description"`
	Passed          bool                   `json:"passed"`
	ExpectedOutcome string                 `json:"expected_outcome"`
	ActualOutcome   string                 `json:"actual_outcome"`
	Details         map[string]interface{} `json:"details"`
	Duration        time.Duration          `json:"duration"`
	Error           string                 `json:"error,omitempty"`
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Name}} - Test Report</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 20px;
            color: #333;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }

        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px;
            text-align: center;
        }

        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
            text-shadow: 0 2px 4px rgba(0,0,0,0.2);
        }

        .header .subtitle {
            font-size: 1.1em;
            opacity: 0.9;
        }

        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 40px;
            background: #f8f9fa;
            border-bottom: 3px solid #e9ecef;
        }

        .summary-card {
            background: white;
            padding: 25px;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            text-align: center;
            transition: transform 0.2s;
        }

        .summary-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        }

        .summary-card .label {
            font-size: 0.9em;
            color: #6c757d;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 10px;
        }

        .summary-card .value {
            font-size: 2.5em;
            font-weight: bold;
            color: #333;
        }

        .summary-card.passed .value {
            color: #28a745;
        }

        .summary-card.failed .value {
            color: #dc3545;
        }

        .summary-card.total .value {
            color: #667eea;
        }

        .summary-card.duration .value {
            font-size: 1.8em;
            color: #6c757d;
        }

        .pass-rate {
            margin-top: 10px;
            font-size: 1.2em;
            padding: 8px;
            border-radius: 4px;
            background: #28a745;
            color: white;
        }

        .pass-rate.warning {
            background: #ffc107;
        }

        .pass-rate.danger {
            background: #dc3545;
        }

        .filters {
            padding: 30px 40px;
            background: white;
            border-bottom: 1px solid #e9ecef;
        }

        .filter-buttons {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
        }

        .filter-btn {
            padding: 10px 20px;
            border: 2px solid #667eea;
            background: white;
            color: #667eea;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.95em;
            font-weight: 600;
            transition: all 0.2s;
        }

        .filter-btn:hover {
            background: #667eea;
            color: white;
        }

        .filter-btn.active {
            background: #667eea;
            color: white;
        }

        .tests {
            padding: 40px;
        }

        .test-category {
            margin-bottom: 40px;
        }

        .category-header {
            font-size: 1.5em;
            font-weight: bold;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 3px solid #667eea;
            color: #667eea;
        }

        .test-card {
            background: white;
            border: 2px solid #e9ecef;
            border-radius: 8px;
            padding: 25px;
            margin-bottom: 20px;
            transition: all 0.2s;
        }

        .test-card:hover {
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
            transform: translateX(5px);
        }

        .test-card.passed {
            border-left: 5px solid #28a745;
        }

        .test-card.failed {
            border-left: 5px solid #dc3545;
            background: #fff5f5;
        }

        .test-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
        }

        .test-name {
            font-size: 1.3em;
            font-weight: bold;
            color: #333;
        }

        .test-badge {
            padding: 6px 16px;
            border-radius: 20px;
            font-size: 0.85em;
            font-weight: bold;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .test-badge.passed {
            background: #28a745;
            color: white;
        }

        .test-badge.failed {
            background: #dc3545;
            color: white;
        }

        .test-description {
            color: #6c757d;
            margin-bottom: 20px;
            line-height: 1.6;
        }

        .test-outcomes {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
            margin-bottom: 20px;
        }

        .outcome-box {
            padding: 15px;
            border-radius: 6px;
            background: #f8f9fa;
        }

        .outcome-label {
            font-size: 0.85em;
            color: #6c757d;
            text-transform: uppercase;
            margin-bottom: 8px;
            font-weight: 600;
        }

        .outcome-value {
            font-size: 0.95em;
            line-height: 1.5;
            color: #333;
        }

        .test-details {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 6px;
            margin-top: 15px;
        }

        .test-details summary {
            cursor: pointer;
            font-weight: bold;
            color: #667eea;
            margin-bottom: 15px;
            user-select: none;
        }

        .test-details summary:hover {
            color: #764ba2;
        }

        .detail-item {
            display: flex;
            padding: 8px 0;
            border-bottom: 1px solid #dee2e6;
        }

        .detail-item:last-child {
            border-bottom: none;
        }

        .detail-key {
            font-weight: 600;
            color: #495057;
            min-width: 180px;
        }

        .detail-value {
            color: #6c757d;
            word-break: break-word;
        }

        .error-box {
            background: #f8d7da;
            border: 1px solid #f5c6cb;
            color: #721c24;
            padding: 15px;
            border-radius: 6px;
            margin-top: 15px;
            font-family: 'Courier New', monospace;
            font-size: 0.9em;
        }

        .duration {
            display: inline-block;
            background: #e9ecef;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 0.85em;
            color: #6c757d;
            margin-left: 10px;
        }

        .footer {
            background: #f8f9fa;
            padding: 30px 40px;
            text-align: center;
            color: #6c757d;
            border-top: 3px solid #e9ecef;
        }

        @media (max-width: 768px) {
            .test-outcomes {
                grid-template-columns: 1fr;
            }

            .summary {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔍 {{.Name}}</h1>
            <div class="subtitle">
                Generated: {{.EndTime.Format "2006-01-02 15:04:05 MST"}}
            </div>
        </div>

        <div class="summary">
            <div class="summary-card total">
                <div class="label">Total Tests</div>
                <div class="value">{{.TotalTests}}</div>
            </div>
            <div class="summary-card passed">
                <div class="label">Passed</div>
                <div class="value">{{.PassedTests}}</div>
            </div>
            <div class="summary-card failed">
                <div class="label">Failed</div>
                <div class="value">{{.FailedTests}}</div>
            </div>
            <div class="summary-card duration">
                <div class="label">Duration</div>
                <div class="value">{{.EndTime.Sub .StartTime | FormatDuration}}</div>
                <div class="pass-rate {{PassRateClass .PassedTests .TotalTests}}">
                    Pass Rate: {{PassRate .PassedTests .TotalTests}}%
                </div>
            </div>
        </div>

        <div class="filters">
            <div class="filter-buttons">
                <button class="filter-btn active" onclick="filterTests('all')">All Tests</button>
                <button class="filter-btn" onclick="filterTests('passed')">✓ Passed Only</button>
                <button class="filter-btn" onclick="filterTests('failed')">✗ Failed Only</button>
                <button class="filter-btn" onclick="filterTests('Deduplication')">Deduplication</button>
                <button class="filter-btn" onclick="filterTests('Correlation')">Correlation</button>
                <button class="filter-btn" onclick="filterTests('Confidence')">Confidence</button>
                <button class="filter-btn" onclick="filterTests('Magnitude')">Magnitude</button>
            </div>
        </div>

        <div class="tests">
            {{range GroupByCategory .Results}}
            <div class="test-category" data-category="{{.Category}}">
                <h2 class="category-header">{{.Category}}</h2>
                {{range .Tests}}
                <div class="test-card {{if .Passed}}passed{{else}}failed{{end}}" data-status="{{if .Passed}}passed{{else}}failed{{end}}">
                    <div class="test-header">
                        <span class="test-name">{{.TestName}}</span>
                        <span>
                            <span class="test-badge {{if .Passed}}passed{{else}}failed{{end}}">
                                {{if .Passed}}✓ Passed{{else}}✗ Failed{{end}}
                            </span>
                            <span class="duration">{{FormatDuration .Duration}}</span>
                        </span>
                    </div>

                    <div class="test-description">{{.Description}}</div>

                    <div class="test-outcomes">
                        <div class="outcome-box">
                            <div class="outcome-label">Expected Outcome</div>
                            <div class="outcome-value">{{.ExpectedOutcome}}</div>
                        </div>
                        <div class="outcome-box">
                            <div class="outcome-label">Actual Outcome</div>
                            <div class="outcome-value">{{.ActualOutcome}}</div>
                        </div>
                    </div>

                    {{if .Details}}
                    <details class="test-details">
                        <summary>📊 View Detailed Results</summary>
                        {{range $key, $value := .Details}}
                        <div class="detail-item">
                            <div class="detail-key">{{$key}}:</div>
                            <div class="detail-value">{{FormatValue $value}}</div>
                        </div>
                        {{end}}
                    </details>
                    {{end}}

                    {{if .Error}}
                    <div class="error-box">
                        <strong>Error:</strong> {{.Error}}
                    </div>
                    {{end}}
                </div>
                {{end}}
            </div>
            {{end}}
        </div>

        <div class="footer">
            <p>OSINT System Integration Test Suite</p>
            <p style="margin-top: 10px; font-size: 0.9em;">
                Testing source deduplication, event correlation, confidence scoring, and magnitude estimation
            </p>
        </div>
    </div>

    <script>
        function filterTests(filter) {
            const cards = document.querySelectorAll('.test-card');
            const categories = document.querySelectorAll('.test-category');
            const buttons = document.querySelectorAll('.filter-btn');

            // Update button states
            buttons.forEach(btn => {
                if (btn.textContent.includes(filter) || (filter === 'all' && btn.textContent.includes('All'))) {
                    btn.classList.add('active');
                } else {
                    btn.classList.remove('active');
                }
            });

            if (filter === 'all') {
                cards.forEach(card => card.style.display = 'block');
                categories.forEach(cat => cat.style.display = 'block');
                return;
            }

            if (filter === 'passed' || filter === 'failed') {
                cards.forEach(card => {
                    card.style.display = card.dataset.status === filter ? 'block' : 'none';
                });
                categories.forEach(cat => {
                    const visibleCards = cat.querySelectorAll('.test-card[data-status="' + filter + '"]');
                    cat.style.display = visibleCards.length > 0 ? 'block' : 'none';
                });
                return;
            }

            // Filter by category
            categories.forEach(cat => {
                const categoryName = cat.dataset.category;
                cat.style.display = categoryName === filter ? 'block' : 'none';
            });
        }
    </script>
</body>
</html>
`

// GenerateHTMLReport generates an HTML test report
func GenerateHTMLReport(suite *TestSuite, filename string) error {
	funcMap := template.FuncMap{
		"FormatDuration": func(d time.Duration) string {
			if d < time.Millisecond {
				return fmt.Sprintf("%dµs", d.Microseconds())
			} else if d < time.Second {
				return fmt.Sprintf("%dms", d.Milliseconds())
			}
			return fmt.Sprintf("%.2fs", d.Seconds())
		},
		"PassRate": func(passed, total int) int {
			if total == 0 {
				return 0
			}
			return (passed * 100) / total
		},
		"PassRateClass": func(passed, total int) string {
			if total == 0 {
				return "danger"
			}
			rate := (passed * 100) / total
			if rate >= 90 {
				return ""
			} else if rate >= 70 {
				return "warning"
			}
			return "danger"
		},
		"GroupByCategory": func(results []TestResult) []CategoryGroup {
			groups := make(map[string][]TestResult)
			order := []string{}

			for _, result := range results {
				if _, exists := groups[result.Category]; !exists {
					order = append(order, result.Category)
				}
				groups[result.Category] = append(groups[result.Category], result)
			}

			categoryGroups := []CategoryGroup{}
			for _, category := range order {
				categoryGroups = append(categoryGroups, CategoryGroup{
					Category: category,
					Tests:    groups[category],
				})
			}
			return categoryGroups
		},
		"FormatValue": func(v interface{}) string {
			switch val := v.(type) {
			case []interface{}:
				if len(val) == 0 {
					return "[]"
				}
				result := "["
				for i, item := range val {
					if i > 0 {
						result += ", "
					}
					result += fmt.Sprintf("%v", item)
				}
				result += "]"
				return result
			case []string:
				if len(val) == 0 {
					return "[]"
				}
				result := "["
				for i, item := range val {
					if i > 0 {
						result += ", "
					}
					result += fmt.Sprintf("\"%s\"", item)
				}
				result += "]"
				return result
			default:
				return fmt.Sprintf("%v", val)
			}
		},
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, suite); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// CategoryGroup groups test results by category
type CategoryGroup struct {
	Category string
	Tests    []TestResult
}
