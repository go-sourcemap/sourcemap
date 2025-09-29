package sourcemap_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-sourcemap/sourcemap"
)

// generateSourceMap creates a synthetic sourcemap with specified characteristics
func generateSourceMap(numSources int, numMappings int) []byte {
	// Generate sources
	sources := make([]string, numSources)
	for i := 0; i < numSources; i++ {
		sources[i] = fmt.Sprintf("src/file%d.js", i)
	}

	// Generate names
	names := []string{"foo", "bar", "baz", "qux", "quux", "corge", "grault", "garply"}

	// Generate mappings string
	// Real sourcemaps have complex VLQ-encoded mappings
	// We'll create a realistic pattern
	var mappingsBuilder strings.Builder
	segmentsPerLine := 50
	linesCount := numMappings / segmentsPerLine

	vlqChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	
	for line := 0; line < linesCount; line++ {
		if line > 0 {
			mappingsBuilder.WriteByte(';')
		}
		
		for seg := 0; seg < segmentsPerLine; seg++ {
			if seg > 0 {
				mappingsBuilder.WriteByte(',')
			}
			
			// Generate a realistic VLQ segment (4-6 characters typical)
			segmentLength := 4 + (line+seg)%3
			for i := 0; i < segmentLength; i++ {
				idx := (line*7 + seg*13 + i*17) % len(vlqChars)
				mappingsBuilder.WriteByte(vlqChars[idx])
			}
		}
	}

	sm := map[string]interface{}{
		"version":        3,
		"file":          "bundle.min.js",
		"sourceRoot":    "",
		"sources":       sources,
		"sourcesContent": make([]string, numSources),
		"names":         names,
		"mappings":      mappingsBuilder.String(),
	}

	data, _ := json.Marshal(sm)
	return data
}

// Benchmark with synthetic data simulating real-world sizes
func BenchmarkSyntheticSizes(b *testing.B) {
	testCases := []struct {
		name        string
		sources     int
		mappings    int
		approxSize  string
	}{
		{"Tiny", 2, 100, "~5KB"},
		{"Small", 5, 500, "~25KB"},
		{"Medium", 10, 2000, "~100KB"},
		{"Large", 50, 10000, "~500KB"},
		{"XLarge", 100, 50000, "~2.5MB"},
		{"XXLarge", 200, 200000, "~10MB"},
	}

	for _, tc := range testCases {
		data := generateSourceMap(tc.sources, tc.mappings)
		
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := sourcemap.Parse("", data)
				if err != nil {
					b.Fatal(err)
				}
			}

			// Calculate and report throughput
			mbPerSec := float64(len(data)*b.N) / b.Elapsed().Seconds() / (1024 * 1024)
			b.ReportMetric(mbPerSec, "MB/s")
		})
	}
}

// Benchmark memory allocations specifically
func BenchmarkMemoryUsage(b *testing.B) {
	sizes := []int{1000, 10000, 50000}

	for _, size := range sizes {
		data := generateSourceMap(10, size)
		
		b.Run(fmt.Sprintf("Mappings_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := sourcemap.Parse("", data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark source lookup performance
func BenchmarkLookupPerformance(b *testing.B) {
	// Generate a medium-sized sourcemap
	data := generateSourceMap(20, 10000)
	smap, err := sourcemap.Parse("", data)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Sequential", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for line := 0; line < 100; line++ {
				smap.Source(line, line)
			}
		}
	})

	b.Run("Random", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 100; j++ {
				line := (j * 137) % 200
				col := (j * 241) % 1000
				smap.Source(line, col)
			}
		}
	})

	b.Run("WorstCase", func(b *testing.B) {
		b.ResetTimer()
		// Always search for the last mapping (worst case for binary search)
		for i := 0; i < b.N; i++ {
			for j := 0; j < 100; j++ {
				smap.Source(199, 999)
			}
		}
	})
}