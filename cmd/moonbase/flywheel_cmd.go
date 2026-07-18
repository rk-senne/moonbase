package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/f5508037/moonbase/internal/pipeline"
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

	for _, e := range entries {
		traceIDs[e.TraceID] = true
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

	fmt.Println("🌙 Moonbase Flywheel — Pipeline Learning Insights")
	fmt.Println()
	fmt.Printf("   Total missions:   %d\n", totalMissions)
	fmt.Printf("   Total phases run: %d\n", totalEntries)
	fmt.Printf("   Rework rate:      %.1f%%\n", reworkRate)
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
}
