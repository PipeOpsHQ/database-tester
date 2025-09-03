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
		"sub":                 func(a, b float64) float64 { return a - b },
		"add":                 func(a, b int64) int64 { return a + b },
		"mul":                 func(a, b int64) int64 { return a * b },
		"div": func(a, b int64) int64 {
			if b == 0 {
				return 0
			}
			return a / b
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
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            background-color: #f5f5f5;
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
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }

        .summary-card {
            background: white;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            border-left: 4px solid #667eea;
        }

        .summary-card h3 {
            font-size: 1.1em;
            color: #666;
            margin-bottom: 10px;
        }

        .summary-card .value {
            font-size: 2em;
            font-weight: bold;
            color: #333;
        }

        .summary-card .unit {
            font-size: 0.9em;
            color: #666;
            margin-left: 5px;
        }

        .section {
            background: white;
            margin-bottom: 30px;
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
        }

        .section-header {
            background: #f8f9fa;
            padding: 20px;
            border-bottom: 1px solid #e9ecef;
        }

        .section-header h2 {
            color: #333;
            font-size: 1.5em;
        }

        .section-content {
            padding: 20px;
        }

        .database-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 20px;
        }

        .database-card {
            border: 1px solid #e9ecef;
            border-radius: 8px;
            padding: 20px;
            background: #fafafa;
        }

        .database-card h3 {
            color: #333;
            margin-bottom: 15px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }

        .status-badge {
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 0.85em;
            font-weight: bold;
        }

        .status-connected {
            background: #d4edda;
            color: #155724;
        }

        .status-disconnected {
            background: #f8d7da;
            color: #721c24;
        }

        .status-unknown {
            background: #fff3cd;
            color: #856404;
        }

        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 15px;
            margin: 15px 0;
        }

        .metric {
            text-align: center;
            padding: 10px;
            background: white;
            border-radius: 6px;
            border: 1px solid #e9ecef;
        }

        .metric-label {
            font-size: 0.85em;
            color: #666;
            margin-bottom: 5px;
        }

        .metric-value {
            font-weight: bold;
            font-size: 1.1em;
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
            <h1>Database Stress Test Report</h1>
            <div class="subtitle">Generated on {{.GeneratedAt.Format "January 2, 2006 at 3:04 PM MST"}}</div>
        </div>

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
                <div class="value">{{printf "%.1f" (sub 100 .Summary.OverallErrorRate)}}<span class="unit">%</span></div>
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
                <div class="database-grid">
                    {{range $db, $result := .TestResults}}
                    <div class="database-card">
                        <h3>
                            {{$db}}
                            {{$status := index $.ConnectionStatus $db}}
                            <span class="status-badge {{GetStatusClass $status}}">{{$status}}</span>
                        </h3>

                        <div class="metrics-grid">
                            <div class="metric">
                                <div class="metric-label">Operations</div>
                                <div class="metric-value">{{$result.TotalOperations}}</div>
                            </div>
                            <div class="metric">
                                <div class="metric-label">Success</div>
                                <div class="metric-value">{{$result.SuccessfulOps}}</div>
                            </div>
                            <div class="metric">
                                <div class="metric-label">Failed</div>
                                <div class="metric-value">{{$result.FailedOps}}</div>
                            </div>
                            <div class="metric">
                                <div class="metric-label">Ops/Sec</div>
                                <div class="metric-value {{GetPerformanceClass $result.OperationsPerSec}}">{{printf "%.1f" $result.OperationsPerSec}}</div>
                            </div>
                            <div class="metric">
                                <div class="metric-label">Error Rate</div>
                                <div class="metric-value {{GetErrorRateClass $result.ErrorRate}}">{{printf "%.2f" $result.ErrorRate}}%</div>
                            </div>
                            <div class="metric">
                                <div class="metric-label">Duration</div>
                                <div class="metric-value">{{FormatDuration $result.TotalDuration}}</div>
                            </div>
                        </div>

                        <div class="latency-chart">
                            <div class="latency-bar" style="height: {{div (mul $result.MinLatency.Nanoseconds 50) (add $result.MaxLatency.Nanoseconds 1)}}px;" title="Min: {{FormatDuration $result.MinLatency}}">Min</div>
                            <div class="latency-bar" style="height: {{div (mul $result.AvgLatency.Nanoseconds 50) (add $result.MaxLatency.Nanoseconds 1)}}px;" title="Avg: {{FormatDuration $result.AvgLatency}}">Avg</div>
                            <div class="latency-bar" style="height: {{div (mul $result.Percentile95.Nanoseconds 50) (add $result.MaxLatency.Nanoseconds 1)}}px;" title="P95: {{FormatDuration $result.Percentile95}}">P95</div>
                            <div class="latency-bar" style="height: {{div (mul $result.Percentile99.Nanoseconds 50) (add $result.MaxLatency.Nanoseconds 1)}}px;" title="P99: {{FormatDuration $result.Percentile99}}">P99</div>
                            <div class="latency-bar" style="height: 50px;" title="Max: {{FormatDuration $result.MaxLatency}}">Max</div>
                        </div>

                        {{if gt (len $result.Errors) 0}}
                        <div class="errors-section">
                            <h4>Errors ({{len $result.Errors}} shown):</h4>
                            <div class="error-list">
                                {{range $result.Errors}}
                                <div class="error-item">{{.}}</div>
                                {{end}}
                            </div>
                        </div>
                        {{end}}
                    </div>
                    {{end}}
                </div>
            </div>
        </div>

        <div class="footer">
            <p>Database Stress Test Report - PipeOps TCP/UDP Port Testing Service</p>
            <p>Generated at {{.GeneratedAt.Format "2006-01-02 15:04:05 MST"}}</p>
        </div>
    </div>

    <script>
        // Add some interactivity
        document.addEventListener('DOMContentLoaded', function() {
            // Add click handlers for metrics
            const metrics = document.querySelectorAll('.metric');
            metrics.forEach(metric => {
                metric.addEventListener('click', function() {
                    this.style.transform = this.style.transform === 'scale(1.05)' ? 'scale(1)' : 'scale(1.05)';
                    this.style.transition = 'transform 0.2s ease';
                });
            });

            // Add hover effects for database cards
            const cards = document.querySelectorAll('.database-card');
            cards.forEach(card => {
                card.addEventListener('mouseenter', function() {
                    this.style.boxShadow = '0 4px 8px rgba(0,0,0,0.15)';
                    this.style.transition = 'box-shadow 0.3s ease';
                });

                card.addEventListener('mouseleave', function() {
                    this.style.boxShadow = 'none';
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
