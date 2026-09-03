package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/rk-senne/moonbase/internal/pipeline"
	"github.com/spf13/cobra"
)

var flywheelCmd = &cobra.Command{
	Use:   "flywheel",
	Short: "Show pipeline learning insights from session history",
	Long:  "Analyze the flywheel log to show patterns in agent performance,\nrework frequency, and improvement opportunities.",
	Run: func(cmd *cobra.Command, args []string) {
		runFlywheel()
	},
}

func runFlywheel() {
	log := pipeline.NewFlywheelLog()
	path := log.Path()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("🌙 No flywheel data yet.")
			fmt.Println()
			fmt.Println("   Run some missions first:")
			fmt.Println("   moonbase mission \"your task here\"")
			fmt.Println()
			fmt.Println("   The flywheel logs every pipeline phase, building")
			fmt.Println("   a picture of where your agents excel and struggle.")
			return
		}
		fmt.Fprintf(os.Stderr, "Error opening flywheel log: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	var entries []pipeline.FlywheelEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry pipeline.FlywheelEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading flywheel log: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("🌙 Flywheel log is empty. Run some missions to generate data.")
		return
	}

	// Compute stats
	traceIDs := make(map[string]bool)
	phaseDurations := make(map[string][]int64)
	phaseFailures := make(map[string]int)
	riskLevels := make(map[string]int)
	totalRework := 0
	totalEntries := len(entries)

	// Group entries by trace for lead-time analysis
	traceEntries := make(map[string][]pipeline.FlywheelEntry)

	for _, e := range entries {
		traceIDs[e.TraceID] = true
		traceEntries[e.TraceID] = append(traceEntries[e.TraceID], e)
		phaseName := fmt.Sprintf("Phase %d (%s)", e.Phase, e.Agent)
		phaseDurations[phaseName] = append(phaseDurations[phaseName], e.DurationMs)

		if e.Outcome == "rework" || e.Outcome == "failed" {
			phaseFailures[phaseName]++
		}
		if e.Outcome == "rework" {
			totalRework++
		}
		if e.RiskLevel != "" {
			riskLevels[e.RiskLevel]++
		}
	}

	totalMissions := len(traceIDs)
	reworkRate := float64(totalRework) / float64(totalEntries) * 100

	// Compute mission lead times and longest phases
	leadTimes, longestPhase := computeLeadTimeInsights(traceEntries)

	fmt.Println("🌙 Moonbase Flywheel — Pipeline Learning Insights")
	fmt.Println()
	fmt.Printf("   Total missions:   %d\n", totalMissions)
	fmt.Printf("   Total phases run: %d\n", totalEntries)
	fmt.Printf("   Rework rate:      %.1f%%\n", reworkRate)
	if leadTimes.count > 0 {
		fmt.Printf("   Avg lead time:    %s\n", leadTimes.avg.Round(time.Millisecond))
		fmt.Printf("   Longest phase:    %s (%s)\n", longestPhase.name, longestPhase.duration.Round(time.Millisecond))
	}
	fmt.Println()

	// Average duration per phase
	fmt.Println("   ⏱  Average Duration per Phase:")
	type phaseStat struct {
		name string
		avg  time.Duration
	}
	var phaseStats []phaseStat
	for name, durations := range phaseDurations {
		var total int64
		for _, d := range durations {
			total += d
		}
		avg := time.Duration(total/int64(len(durations))) * time.Millisecond
		phaseStats = append(phaseStats, phaseStat{name, avg})
	}
	sort.Slice(phaseStats, func(i, j int) bool {
		return phaseStats[i].name < phaseStats[j].name
	})
	for _, ps := range phaseStats {
		fmt.Printf("      %-30s %s\n", ps.name, ps.avg)
	}
	fmt.Println()

	// Risk levels
	if len(riskLevels) > 0 {
		fmt.Println("   🎯 Risk Level Distribution:")
		type riskStat struct {
			level string
			count int
		}
		var risks []riskStat
		for level, count := range riskLevels {
			risks = append(risks, riskStat{level, count})
		}
		sort.Slice(risks, func(i, j int) bool {
			return risks[i].count > risks[j].count
		})
		for _, r := range risks {
			fmt.Printf("      %-12s %d (%.0f%%)\n", r.level, r.count, float64(r.count)/float64(totalEntries)*100)
		}
		fmt.Println()
	}

	// Phases that fail most often
	if len(phaseFailures) > 0 {
		fmt.Println("   ⚠️  Phases with Most Failures/Rework:")
		type failStat struct {
			name  string
			count int
		}
		var fails []failStat
		for name, count := range phaseFailures {
			fails = append(fails, failStat{name, count})
		}
		sort.Slice(fails, func(i, j int) bool {
			return fails[i].count > fails[j].count
		})
		for _, f := range fails {
			fmt.Printf("      %-30s %d occurrence(s)\n", f.name, f.count)
		}
		fmt.Println()
	}

	fmt.Printf("   📁 Log: %s\n", path)

	// Token & Cost section (AC-5: graceful degradation if no token data)
	displayTokenCostInsights(entries)
}

// leadTimeStats holds aggregated mission lead-time metrics.
type leadTimeStats struct {
	avg   time.Duration
	count int
}

// longestPhaseInfo holds the phase with the longest average duration.
type longestPhaseInfo struct {
	name     string
	duration time.Duration
}

// computeLeadTimeInsights computes average mission lead time and the longest phase.
// Lead time per mission = (last entry timestamp + its duration) - first entry timestamp.
func computeLeadTimeInsights(traceEntries map[string][]pipeline.FlywheelEntry) (leadTimeStats, longestPhaseInfo) {
	var totalLeadTime time.Duration
	missionCount := 0

	// Per-phase total duration for finding the longest
	phaseTotals := make(map[string]int64)
	phaseCounts := make(map[string]int)

	for _, entries := range traceEntries {
		if len(entries) == 0 {
			continue
		}

		// Sort by timestamp to find first and last
		first := entries[0]
		last := entries[0]
		for _, e := range entries[1:] {
			if e.Timestamp.Before(first.Timestamp) {
				first = e
			}
			if e.Timestamp.After(last.Timestamp) || (e.Timestamp.Equal(last.Timestamp) && e.DurationMs > last.DurationMs) {
				last = e
			}
		}

		leadTime := last.Timestamp.Sub(first.Timestamp) + time.Duration(last.DurationMs)*time.Millisecond
		if leadTime > 0 {
			totalLeadTime += leadTime
			missionCount++
		}

		// Accumulate per-phase durations
		for _, e := range entries {
			phaseName := fmt.Sprintf("Phase %d (%s)", e.Phase, e.Agent)
			phaseTotals[phaseName] += e.DurationMs
			phaseCounts[phaseName]++
		}
	}

	var stats leadTimeStats
	if missionCount > 0 {
		stats.avg = totalLeadTime / time.Duration(missionCount)
		stats.count = missionCount
	}

	var longest longestPhaseInfo
	for name, total := range phaseTotals {
		avg := time.Duration(total/int64(phaseCounts[name])) * time.Millisecond
		if avg > longest.duration {
			longest.name = name
			longest.duration = avg
		}
	}

	return stats, longest
}

// displayTokenCostInsights shows token/cost aggregation if any entries have token data.
// Graceful degradation: if no entries have token data, the section is skipped entirely.
func displayTokenCostInsights(entries []pipeline.FlywheelEntry) {
	// Filter entries that have token data
	var withTokens []pipeline.FlywheelEntry
	for _, e := range entries {
		if e.TotalTokens > 0 {
			withTokens = append(withTokens, e)
		}
	}

	if len(withTokens) == 0 {
		fmt.Println()
		fmt.Println("   💰 Token & Cost: (no usage data)")
		return
	}

	// Totals
	var totalPrompt, totalCompletion int
	var totalCost float64
	for _, e := range withTokens {
		totalPrompt += e.PromptTokens
		totalCompletion += e.CompletionTokens
		totalCost += e.EstimatedCostUSD
	}

	fmt.Println()
	fmt.Println("   💰 Token & Cost Summary:")
	fmt.Printf("      Total tokens:     %s prompt / %s completion\n",
		formatTokenCount(totalPrompt), formatTokenCount(totalCompletion))
	fmt.Printf("      Total est. cost:  $%.2f\n", totalCost)
	fmt.Println()

	// Per-agent breakdown
	type agentStats struct {
		name            string
		totalPrompt     int
		totalCompletion int
		totalCost       float64
		count           int
	}
	agentMap := make(map[string]*agentStats)
	for _, e := range withTokens {
		s, ok := agentMap[e.Agent]
		if !ok {
			s = &agentStats{name: e.Agent}
			agentMap[e.Agent] = s
		}
		s.totalPrompt += e.PromptTokens
		s.totalCompletion += e.CompletionTokens
		s.totalCost += e.EstimatedCostUSD
		s.count++
	}

	// Sort agents by cost descending
	var agentList []*agentStats
	for _, s := range agentMap {
		agentList = append(agentList, s)
	}
	sort.Slice(agentList, func(i, j int) bool {
		return agentList[i].totalCost/float64(agentList[i].count) > agentList[j].totalCost/float64(agentList[j].count)
	})

	fmt.Println("   💰 Cost per Agent (avg per invocation):")
	for _, s := range agentList {
		avgCost := s.totalCost / float64(s.count)
		avgPrompt := s.totalPrompt / s.count
		avgCompletion := s.totalCompletion / s.count
		fmt.Printf("      %-30s $%.2f  (%s prompt, %s completion)\n",
			s.name, avgCost, formatTokenCount(avgPrompt), formatTokenCount(avgCompletion))
	}
	fmt.Println()

	// Per-mission breakdown (grouped by trace_id)
	type missionStats struct {
		traceID string
		tokens  int
		cost    float64
	}
	missionMap := make(map[string]*missionStats)
	for _, e := range withTokens {
		s, ok := missionMap[e.TraceID]
		if !ok {
			s = &missionStats{traceID: e.TraceID}
			missionMap[e.TraceID] = s
		}
		s.tokens += e.TotalTokens
		s.cost += e.EstimatedCostUSD
	}

	var missionList []*missionStats
	for _, s := range missionMap {
		missionList = append(missionList, s)
	}

	if len(missionList) > 0 {
		var totalMissionTokens int
		var totalMissionCost float64
		var maxCost float64
		var maxCostTrace string
		var maxTokens int

		for _, m := range missionList {
			totalMissionTokens += m.tokens
			totalMissionCost += m.cost
			if m.cost > maxCost {
				maxCost = m.cost
				maxCostTrace = m.traceID
				maxTokens = m.tokens
			}
		}

		missionCount := len(missionList)
		avgTokens := totalMissionTokens / missionCount
		avgCost := totalMissionCost / float64(missionCount)

		fmt.Println("   💰 Cost per Mission:")
		fmt.Printf("      Avg tokens/mission:  %s\n", formatTokenCount(avgTokens))
		fmt.Printf("      Avg cost/mission:    $%.2f\n", avgCost)
		if maxCostTrace != "" {
			fmt.Printf("      Most expensive:      %s ($%.2f, %s tokens)\n",
				truncateTrace(maxCostTrace), maxCost, formatTokenCount(maxTokens))
		}
		fmt.Println()
	}

	// Cost-heavy phase detection
	type phaseStats struct {
		phase   int
		agent   string
		avgCost float64
		count   int
		total   float64
	}
	phaseMap := make(map[int]*phaseStats)
	for _, e := range withTokens {
		s, ok := phaseMap[e.Phase]
		if !ok {
			s = &phaseStats{phase: e.Phase, agent: e.Agent}
			phaseMap[e.Phase] = s
		}
		s.total += e.EstimatedCostUSD
		s.count++
	}

	var costliestPhase *phaseStats
	for _, s := range phaseMap {
		s.avgCost = s.total / float64(s.count)
		if costliestPhase == nil || s.avgCost > costliestPhase.avgCost {
			costliestPhase = s
		}
	}

	if costliestPhase != nil && totalCost > 0 {
		pct := (costliestPhase.total / totalCost) * 100
		fmt.Println("   💰 Cost-Heavy Phase:")
		fmt.Printf("      Phase %d (%s) — avg $%.2f/invocation (%.0f%% of total cost)\n",
			costliestPhase.phase, costliestPhase.agent, costliestPhase.avgCost, pct)
		fmt.Println()
	}
}

// formatTokenCount formats large token counts with K/M suffixes.
func formatTokenCount(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%dK", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

// truncateTrace shortens a trace ID for display.
func truncateTrace(traceID string) string {
	if len(traceID) > 20 {
		return traceID[:20] + "..."
	}
	return traceID
}
