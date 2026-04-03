package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/i2y/dark"
)

type DashboardArgs struct {
	Period string `json:"period" jsonschema:"description=Time period: day, week, or month"`
}

type StatsArgs struct {
	Metric string `json:"metric" jsonschema:"description=Metric name to query"`
}

func main() {
	ctx := context.Background()

	mcpApp, err := dark.NewMCPApp("analytics-server", "1.0.0",
		dark.WithMCPTemplateDir("views"),
		dark.WithMCPPoolSize(2),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer mcpApp.Close()

	// UI tool: interactive analytics dashboard
	if err := dark.AddUITool(mcpApp, "dashboard", dark.UIToolDef{
		Description: "Show an interactive analytics dashboard for the given time period",
		Component:   "mcp/dashboard.tsx",
		Title:       "Analytics Dashboard",
	}, func(ctx context.Context, args DashboardArgs) (map[string]any, error) {
		period := args.Period
		if period == "" {
			period = "week"
		}
		return map[string]any{
			"period":    period,
			"visitors":  rand.Intn(10000) + 1000,
			"pageViews": rand.Intn(50000) + 5000,
			"topPages": []map[string]any{
				{"path": "/", "views": rand.Intn(5000) + 1000},
				{"path": "/docs", "views": rand.Intn(3000) + 500},
				{"path": "/blog", "views": rand.Intn(2000) + 300},
				{"path": "/pricing", "views": rand.Intn(1000) + 100},
			},
		}, nil
	}); err != nil {
		log.Fatal(err)
	}

	// Text tool: plain text statistics
	dark.AddTextTool(mcpApp, "stats", "Get summary statistics for a given metric as plain text",
		func(ctx context.Context, args StatsArgs) (string, error) {
			metric := args.Metric
			if metric == "" {
				metric = "visitors"
			}
			value := rand.Intn(10000) + 100
			return fmt.Sprintf("Metric: %s\nValue: %d\nPeriod: last 7 days", metric, value), nil
		})

	fmt.Fprintln(os.Stderr, "analytics-server MCP app running on stdio...")
	if err := mcpApp.RunStdio(ctx); err != nil {
		log.Fatal(err)
	}
}
