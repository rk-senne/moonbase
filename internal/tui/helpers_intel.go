package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func (a *App) addIntel(format string, args ...any) {
	entry := IntelEntry{
		Time:    time.Now().Format("15:04"),
		Message: fmt.Sprintf(format, args...),
	}
	a.intel = append(a.intel, entry)
	if len(a.intel) > maxIntelEntries {
		a.intel = a.intel[len(a.intel)-maxIntelEntries:]
	}
	a.anim.TriggerIntelFlash()
}

func (a *App) filterAgents() {
	query := strings.ToLower(a.searchInput.Value())
	if query == "" {
		a.filtered = nil
		return
	}

	type scored struct {
		index int
		score int
	}

	var matches []scored
	for i, agent := range a.registry.All() {
		name := strings.ToLower(agent.Name)
		desc := strings.ToLower(agent.Description)

		nameScore, nameHit := fuzzyMatch(query, name)
		descScore, descHit := fuzzyMatch(query, desc)

		if nameHit || descHit {
			best := nameScore
			if descScore > best {
				best = descScore
			}
			matches = append(matches, scored{index: i, score: best})
		}
	}

	// Stable sort by score descending — ties preserve registry order.
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	a.filtered = nil
	for _, m := range matches {
		a.filtered = append(a.filtered, m.index)
	}
}

func (a App) gitStatus() string {
	return a.system.GitStatus()
}

func (a App) uptime() string {
	d := time.Since(a.startTime)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (a App) detectedBackends() string {
	var names []string
	for _, b := range a.backends {
		if b.Available() {
			names = append(names, b.Name())
		}
	}
	if len(names) == 0 {
		return "clipboard only"
	}
	return strings.Join(names, ", ")
}
