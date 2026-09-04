package history

import (
	"fmt"
	"testing"
)

// Save reads and rewrites the entire history file on every call, so the cost of
// one save grows with the number of missions already stored. These benchmarks
// measure that directly rather than reasoning about it.
//
// Run with:
//
//	go test -bench BenchmarkSave -benchtime 20x ./internal/history/
func BenchmarkSave_GrowingHistory(b *testing.B) {
	for _, existing := range []int{0, 100, 1000, 5000} {
		b.Run(fmt.Sprintf("existing=%d", existing), func(b *testing.B) {
			dir := b.TempDir()
			b.Setenv("HOME", dir)
			b.Setenv("XDG_CONFIG_HOME", dir)

			// Seed the file with `existing` missions.
			seed := make([]Mission, 0, existing)
			for i := 0; i < existing; i++ {
				seed = append(seed, Mission{
					ID:            i + 1,
					Task:          fmt.Sprintf("seeded mission %d", i),
					SchemaVersion: currentMissionVersion,
				})
			}
			if existing > 0 {
				if err := writeHistory(seed); err != nil {
					b.Fatalf("seeding: %v", err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Save(Mission{Task: "benchmark mission"}); err != nil {
					b.Fatalf("save: %v", err)
				}
			}
		})
	}
}
