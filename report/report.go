package report

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"tcp-stress-test/stresstester"
)

// ReportData holds all data needed for generating reports
type ReportData struct {
	GeneratedAt      time.Time
	TestResults      map[string]*stresstester.TestResult
	ConnectionStatus map[string]string
	Summary          *ReportSummary
}

// ReportSummary holds summary statistics
type ReportSummary struct {
	TotalDatabases     int
	ConnectedDatabases int
	TotalOperations    int64
	TotalSuccessful    int64
	TotalFailed        int64
	OverallErrorRate   float64
	AverageOpsPerSec   float64
	FastestDatabase    string
	SlowestDatabase    string
	MostReliable       string
}

// ReportGenerator generates HTML reports
type ReportGenerator struct {
	template *template.Template
}

// NewReportGenerator creates a new report generator
func NewReportGenerator() *ReportGenerator {
	funcMap := template.FuncMap{
		"FormatDuration":      FormatDuration,
		"GetStatusClass":      GetStatusClass,
		"GetPerformanceClass": GetPerformanceClass,
		"GetErrorRateClass":   GetErrorRateClass,
		"sub": func(a, b interface{}) interface{} {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va - vb
				}
			case float64:
				if vb, ok := b.(float64); ok {
					return va - vb
				}
			}
			return 0
		},
		"add": func(a, b interface{}) interface{} {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va + vb
				}
			case int64:
				if vb, ok := b.(int64); ok {
					return va + vb
				}
			}
			return 0
		},
		"mul": func(a, b interface{}) interface{} {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va * vb
				}
			case int64:
				if vb, ok := b.(int64); ok {
					return va * vb
				}
			}
			return 0
		},
		"div": func(a, b interface{}) interface{} {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok && vb != 0 {
					return va / vb
				}
			case int64:
				if vb, ok := b.(int64); ok && vb != 0 {
					return va / vb
				}
			}
			return 0
		},
		"len": func(v interface{}) int {
			switch s := v.(type) {
			case []string:
				return len(s)
			case string:
				return len(s)
			default:
				return 0
			}
		},
		"gt": func(a, b interface{}) bool {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va > vb
				}
			case float64:
				if vb, ok := b.(float64); ok {
					return va > vb
				}
			}
			return false
		},
		"lt": func(a, b interface{}) bool {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va < vb
				}
			case float64:
				if vb, ok := b.(float64); ok {
					return va < vb
				}
			}
			return false
		},
	}

	return &ReportGenerator{
		template: template.Must(template.New("report").Funcs(funcMap).Parse(htmlTemplate)),
	}
}

// GenerateHTML generates an HTML report from test results
func (rg *ReportGenerator) GenerateHTML(results map[string]*stresstester.TestResult, connectionStatus map[string]string) (string, error) {
	data := &ReportData{
		GeneratedAt:      time.Now(),
		TestResults:      results,
		ConnectionStatus: connectionStatus,
		Summary:          rg.generateSummary(results, connectionStatus),
	}

	var result strings.Builder
	err := rg.template.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("failed to generate HTML report: %v", err)
	}

	return result.String(), nil
}

// generateSummary creates a summary of all test results
func (rg *ReportGenerator) generateSummary(results map[string]*stresstester.TestResult, connectionStatus map[string]string) *ReportSummary {
	summary := &ReportSummary{
		TotalDatabases: len(connectionStatus),
	}

	var totalOpsPerSec float64
	var fastestOps float64
	var slowestOps float64 = 999999999
	var bestErrorRate float64 = 100

	// Count connected databases
	for _, status := range connectionStatus {
		if status == "Connected" {
			summary.ConnectedDatabases++
		}
	}

	// Analyze test results
	for dbName, result := range results {
		summary.TotalOperations += result.TotalOperations
		summary.TotalSuccessful += result.SuccessfulOps
		summary.TotalFailed += result.FailedOps
		totalOpsPerSec += result.OperationsPerSec

		// Find fastest database
		if result.OperationsPerSec > fastestOps {
			fastestOps = result.OperationsPerSec
			summary.FastestDatabase = dbName
		}

		// Find slowest database
		if result.OperationsPerSec < slowestOps && result.OperationsPerSec > 0 {
			slowestOps = result.OperationsPerSec
			summary.SlowestDatabase = dbName
		}

		// Find most reliable database
		if result.ErrorRate < bestErrorRate {
			bestErrorRate = result.ErrorRate
			summary.MostReliable = dbName
		}
	}

	// Calculate overall metrics
	if summary.TotalOperations > 0 {
		summary.OverallErrorRate = (float64(summary.TotalFailed) / float64(summary.TotalOperations)) * 100
	}

	if len(results) > 0 {
		summary.AverageOpsPerSec = totalOpsPerSec / float64(len(results))
	}

	return summary
}

// FormatDuration formats a duration for display
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	}
	return d.String()
}

// GetStatusClass returns CSS class based on status
func GetStatusClass(status string) string {
	switch status {
	case "Connected":
		return "status-connected"
	case "Disconnected":
		return "status-disconnected"
	default:
		return "status-unknown"
	}
}

// GetPerformanceClass returns CSS class based on operations per second
func GetPerformanceClass(opsPerSec float64) string {
	if opsPerSec >= 100 {
		return "perf-excellent"
	} else if opsPerSec >= 50 {
		return "perf-good"
	} else if opsPerSec >= 10 {
		return "perf-average"
	}
	return "perf-poor"
}

// GetErrorRateClass returns CSS class based on error rate
func GetErrorRateClass(errorRate float64) string {
	if errorRate == 0 {
		return "error-none"
	} else if errorRate < 1 {
		return "error-low"
	} else if errorRate < 5 {
		return "error-medium"
	}
	return "error-high"
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Database Stress Test Report</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Courier New', monospace;
            line-height: 1.6;
            color: #00ff00;
            background-color: #000;
            padding: 20px;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }

        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }

        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }

        .header .subtitle {
            font-size: 1.2em;
            opacity: 0.9;
        }

        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
            gap: 10px;
            margin-bottom: 20px;
        }

        .summary-card {
            background: #111;
            padding: 15px;
            border: 1px solid #333;
            text-align: left;
        }

        .summary-card h3 {
            color: #888;
            font-size: 0.8em;
            margin-bottom: 5px;
            text-transform: uppercase;
        }

        .summary-card .value {
            color: #0ff;
            font-size: 1.5em;
            font-weight: normal;
        }

        .summary-card .unit {
            font-size: 0.6em;
            color: #666;
        }

        .section {
            background: #000;
            border: 1px solid #333;
            margin-bottom: 20px;
        }

        .section-header {
            background: #111;
            color: #00ff00;
            padding: 10px;
            border-bottom: 1px solid #333;
        }

        .section-header h2 {
            font-size: 1.2em;
            text-transform: uppercase;
        }

        .section-content {
            padding: 15px;
        }

        .database-grid {
            display: grid;
            grid-template-columns: 1fr;
            gap: 15px;
        }

        .database-card {
            background: #111;
            padding: 15px;
            border: 1px solid #333;
        }

        .database-card h3 {
            color: #0ff;
            margin-bottom: 10px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            text-transform: uppercase;
            font-size: 1em;
        }

        .status-badge {
            padding: 2px 6px;
            border: 1px solid;
            font-size: 0.8em;
            text-transform: uppercase;
        }

        .status-connected {
            border-color: #00ff00;
            color: #00ff00;
        }

        .status-disconnected {
            border-color: #ff0000;
            color: #ff0000;
        }

        .status-unknown {
            border-color: #ffff00;
            color: #ffff00;
        }

        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
            gap: 5px;
            margin-top: 10px;
        }

        .metric {
            padding: 5px;
            border-left: 2px solid #333;
            padding-left: 10px;
        }

        .metric-label {
            font-size: 0.8em;
            color: #888;
            margin-bottom: 2px;
        }

        .metric-value {
            font-size: 1em;
            color: #0ff;
        }

        .perf-excellent { color: #28a745; }
        .perf-good { color: #17a2b8; }
        .perf-average { color: #ffc107; }
        .perf-poor { color: #dc3545; }

        .error-none { color: #28a745; }
        .error-low { color: #17a2b8; }
        .error-medium { color: #ffc107; }
        .error-high { color: #dc3545; }

        .latency-chart {
            display: flex;
            align-items: end;
            justify-content: space-between;
            height: 60px;
            margin: 15px 0;
            padding: 0 10px;
            background: white;
            border-radius: 6px;
            border: 1px solid #e9ecef;
        }

        .latency-bar {
            background: linear-gradient(to top, #667eea, #764ba2);
            border-radius: 2px 2px 0 0;
            min-width: 8px;
            margin: 0 2px;
            display: flex;
            align-items: end;
            justify-content: center;
            color: white;
            font-size: 0.7em;
            padding-bottom: 2px;
        }

        .errors-section {
            margin-top: 15px;
        }

        .error-list {
            background: #fff5f5;
            border: 1px solid #feb2b2;
            border-radius: 6px;
            padding: 15px;
            max-height: 200px;
            overflow-y: auto;
        }

        .error-item {
            padding: 8px 0;
            border-bottom: 1px solid #fed7d7;
            font-family: 'Courier New', monospace;
            font-size: 0.9em;
            color: #c53030;
        }

        .error-item:last-child {
            border-bottom: none;
        }

        .connection-status {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
        }

        .connection-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 15px;
            background: #fafafa;
            border-radius: 6px;
            border: 1px solid #e9ecef;
        }

        .footer {
            text-align: center;
            color: #666;
            margin-top: 30px;
            padding: 20px;
            background: white;
            border-radius: 10px;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
        }

        .summary-text {
            background: #000;
            border: 1px solid #333;
            padding: 20px;
            margin: 20px 0;
            color: #0ff;
            font-family: 'Courier New', monospace;
            font-size: 14px;
            line-height: 1.4;
            white-space: pre;
            overflow-x: auto;
        }

        @media (max-width: 768px) {
            .summary-grid {
                grid-template-columns: 1fr;
            }

            .database-grid {
                grid-template-columns: 1fr;
            }

            .metrics-grid {
                grid-template-columns: repeat(2, 1fr);
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>DATABASE STRESS TEST REPORT</h1>
            <div class="subtitle">Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05 MST"}}</div>
        </div>

        <pre class="summary-text">
================================================================================
QUICK SUMMARY
================================================================================
Test Duration:    {{if .TestResults}}{{range $db, $result := .TestResults}}{{if eq $db $.Summary.FastestDatabase}}{{.TotalDuration}}{{end}}{{end}}{{else}}N/A{{end}}
Total Databases:  {{.Summary.TotalDatabases}} ({{.Summary.ConnectedDatabases}} connected)
Total Operations: {{.Summary.TotalOperations}}
Success Rate:     {{printf "%.1f" (sub 100.0 .Summary.OverallErrorRate)}}%
Avg Ops/Second:   {{printf "%.1f" .Summary.AverageOpsPerSec}}

BEST PERFORMERS:
• Fastest:        {{if .Summary.FastestDatabase}}{{.Summary.FastestDatabase}}{{else}}N/A{{end}}
• Most Reliable:  {{if .Summary.MostReliable}}{{.Summary.MostReliable}}{{else}}N/A{{end}}
• Slowest:        {{if .Summary.SlowestDatabase}}{{.Summary.SlowestDatabase}}{{else}}N/A{{end}}
================================================================================
        </pre>

        <div class="summary-grid">
            <div class="summary-card">
                <h3>Total Databases</h3>
                <div class="value">{{.Summary.TotalDatabases}}</div>
            </div>
            <div class="summary-card">
                <h3>Connected</h3>
                <div class="value">{{.Summary.ConnectedDatabases}}</div>
            </div>
            <div class="summary-card">
                <h3>Total Operations</h3>
                <div class="value">{{.Summary.TotalOperations}}</div>
            </div>
            <div class="summary-card">
                <h3>Success Rate</h3>
                <div class="value">{{printf "%.1f" (sub 100.0 .Summary.OverallErrorRate)}}<span class="unit">%</span></div>
            </div>
            <div class="summary-card">
                <h3>Avg Ops/Sec</h3>
                <div class="value">{{printf "%.1f" .Summary.AverageOpsPerSec}}</div>
            </div>
            <div class="summary-card">
                <h3>Fastest DB</h3>
                <div class="value" style="font-size: 1.2em;">{{.Summary.FastestDatabase}}</div>
            </div>
        </div>

        <div class="section">
            <div class="section-header">
                <h2>Connection Status</h2>
            </div>
            <div class="section-content">
                <div class="connection-status">
                    {{range $db, $status := .ConnectionStatus}}
                    <div class="connection-item">
                        <strong>{{$db}}</strong>
                        <span class="status-badge {{GetStatusClass $status}}">{{$status}}</span>
                    </div>
                    {{end}}
                </div>
            </div>
        </div>

        <div class="section">
            <div class="section-header">
                <h2>Detailed Test Results</h2>
            </div>
            <div class="section-content">
                <pre style="color: #0ff; font-size: 12px; line-height: 1.4;">
+------------------+------------+------------+------------+------------+------------+------------+
| DATABASE         | STATUS     | OPERATIONS | SUCCESS    | FAILED     | OPS/SEC    | ERROR %    |
+------------------+------------+------------+------------+------------+------------+------------+
{{range $db, $result := .TestResults -}}
| {{printf "%-16s" $db}} | {{$status := index $.ConnectionStatus $db}}{{printf "%-10s" $status}} | {{printf "%10d" $result.TotalOperations}} | {{printf "%10d" $result.SuccessfulOps}} | {{printf "%10d" $result.FailedOps}} | {{printf "%10.1f" $result.OperationsPerSec}} | {{printf "%9.1f%%" $result.ErrorRate}} |
{{end -}}
+------------------+------------+------------+------------+------------+------------+------------+

LATENCY STATISTICS:
{{range $db, $result := .TestResults}}
{{$db}}:
  • Average: {{FormatDuration $result.AvgLatency}}
  • Min:     {{FormatDuration $result.MinLatency}}
  • Max:     {{FormatDuration $result.MaxLatency}}
  • P95:     {{FormatDuration $result.Percentile95}}
  • P99:     {{FormatDuration $result.Percentile99}}
{{end}}
                </pre>
            </div>
        </div>

        {{range $db, $result := .TestResults}}
        {{if $result.Errors}}
        <div class="section">
            <div class="section-header">
                <h2>{{$db}} - Error Details</h2>
            </div>
            <div class="section-content">
                <pre style="color: #ff6666; font-size: 12px; max-height: 200px; overflow-y: auto;">{{range $result.Errors}}{{.}}
{{end}}</pre>
            </div>
        </div>
        {{end}}
        {{end}}
    </div>

        <div class="footer">
            <p>Database Stress Test Report - PipeOps TCP/UDP Port Testing Service</p>
            <p>Generated at {{.GeneratedAt.Format "2006-01-02 15:04:05 MST"}}</p>
        </div>
    </div>

    <script>
        // Simple copy functionality for terminal output
        document.addEventListener('DOMContentLoaded', function() {
            const pres = document.querySelectorAll('pre');
            pres.forEach(pre => {
                pre.style.cursor = 'pointer';
                pre.title = 'Click to select all';
                pre.addEventListener('click', function() {
                    const selection = window.getSelection();
                    const range = document.createRange();
                    range.selectNodeContents(this);
                    selection.removeAllRanges();
                    selection.addRange(range);
                });
            });

            // Auto-refresh functionality (commented out for now)
            /*
            setInterval(function() {
                if (confirm('Refresh the report with latest data?')) {
                    window.location.reload();
                }
            }, 300000); // 5 minutes
            */
        });
    </script>
</body>
</html>
`
