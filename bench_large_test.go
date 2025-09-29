package sourcemap_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-sourcemap/sourcemap"
)

// Benchmark data structures
type benchmarkSourceMap struct {
	name     string
	url      string
	size     string
	data     []byte
	gzipData []byte
}

var benchmarkMaps = []benchmarkSourceMap{
	{
		name: "Small",
		url:  "https://cdn.jsdelivr.net/npm/preact@10.19.3/dist/preact.min.js.map",
		size: "~10KB",
	},
	{
		name: "jQuery",
		url:  "https://code.jquery.com/jquery-3.7.1.min.map",
		size: "~135KB",
	},
	{
		name: "Angular",
		url:  "https://cdn.jsdelivr.net/npm/@angular/core@17.0.0/fesm2022/core.min.js.map",
		size: "~2.7MB",
	},
	{
		name: "ReactDOM",
		url:  "https://unpkg.com/react-dom@18.2.0/umd/react-dom.production.min.js.map",
		size: "~1MB",
	},
	{
		name: "VueJS",
		url:  "https://cdn.jsdelivr.net/npm/vue@3.4.15/dist/vue.global.prod.js.map",
		size: "~500KB",
	},
}

func init() {
	// Try to load cached sourcemaps or download them
	cacheDir := filepath.Join(os.TempDir(), "sourcemap-bench-cache")
	os.MkdirAll(cacheDir, 0755)

	for i := range benchmarkMaps {
		sm := &benchmarkMaps[i]
		cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s.map", sm.name))
		
		// Try to load from cache first
		data, err := os.ReadFile(cacheFile)
		if err == nil {
			sm.data = data
			// Create gzip version
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			gz.Write(data)
			gz.Close()
			sm.gzipData = buf.Bytes()
			continue
		}

		// Download if not cached
		fmt.Printf("Downloading %s sourcemap (%s)...\n", sm.name, sm.size)
		resp, err := http.Get(sm.url)
		if err != nil {
			fmt.Printf("Warning: Failed to download %s: %v\n", sm.name, err)
			continue
		}
		defer resp.Body.Close()

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Warning: Failed to read %s: %v\n", sm.name, err)
			continue
		}

		sm.data = data
		
		// Cache for future runs
		os.WriteFile(cacheFile, data, 0644)
		
		// Create gzip version
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write(data)
		gz.Close()
		sm.gzipData = buf.Bytes()
	}
}

// Benchmarks for parsing different sizes of sourcemaps
func BenchmarkParseSizes(b *testing.B) {
	for _, sm := range benchmarkMaps {
		if sm.data == nil {
			continue
		}
		
		b.Run(sm.name, func(b *testing.B) {
			b.SetBytes(int64(len(sm.data)))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_, err := sourcemap.Parse("", sm.data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark memory allocations
func BenchmarkParseAllocs(b *testing.B) {
	for _, sm := range benchmarkMaps {
		if sm.data == nil {
			continue
		}
		
		b.Run(sm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_, err := sourcemap.Parse("", sm.data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark source lookups after parsing
func BenchmarkSourceLookup(b *testing.B) {
	for _, sm := range benchmarkMaps {
		if sm.data == nil {
			continue
		}
		
		smap, err := sourcemap.Parse("", sm.data)
		if err != nil {
			b.Skip("Failed to parse:", err)
			continue
		}
		
		b.Run(sm.name, func(b *testing.B) {
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				// Lookup at various positions
				for j := 0; j < 100; j++ {
					line := (j * 137) % 1000  // Pseudo-random line
					col := (j * 241) % 500    // Pseudo-random column
					smap.Source(line, col)
				}
			}
		})
	}
}

// Benchmark throughput (MB/s)
func BenchmarkThroughput(b *testing.B) {
	for _, sm := range benchmarkMaps {
		if sm.data == nil {
			continue
		}
		
		b.Run(sm.name, func(b *testing.B) {
			b.SetBytes(int64(len(sm.data)))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_, err := sourcemap.Parse("", sm.data)
				if err != nil {
					b.Fatal(err)
				}
			}
			
			// Report throughput
			mbPerSec := float64(len(sm.data)*b.N) / b.Elapsed().Seconds() / (1024 * 1024)
			b.ReportMetric(mbPerSec, "MB/s")
		})
	}
}

// Benchmark parsing with gzipped data (common in real-world)
func BenchmarkParseGzipped(b *testing.B) {
	for _, sm := range benchmarkMaps {
		if sm.gzipData == nil {
			continue
		}
		
		b.Run(sm.name, func(b *testing.B) {
			b.SetBytes(int64(len(sm.data))) // Report original size for fair comparison
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				reader, err := gzip.NewReader(bytes.NewReader(sm.gzipData))
				if err != nil {
					b.Fatal(err)
				}
				
				data, err := io.ReadAll(reader)
				if err != nil {
					b.Fatal(err)
				}
				reader.Close()
				
				_, err = sourcemap.Parse("", data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Generate a synthetic large sourcemap for stress testing
func generateLargeSourceMap(numMappings int) []byte {
	var mappings []string
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	
	for i := 0; i < numMappings; i++ {
		// Generate a pseudo-random VLQ sequence
		length := 2 + (i % 4)
		var mapping string
		for j := 0; j < length; j++ {
			mapping += string(chars[(i*7+j*13)%len(chars)])
		}
		mappings = append(mappings, mapping)
		
		// Add line breaks periodically
		if i%100 == 99 {
			mappings = append(mappings, ";")
		} else if i%10 == 9 {
			mappings = append(mappings, ",")
		}
	}
	
	sm := map[string]interface{}{
		"version": 3,
		"file":    "generated.js",
		"sources": []string{"src/file1.js", "src/file2.js", "src/file3.js"},
		"names":   []string{"foo", "bar", "baz"},
		"mappings": base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v", mappings))),
	}
	
	data, _ := json.Marshal(sm)
	return data
}

// Benchmark with synthetic data of various sizes
func BenchmarkSynthetic(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	
	for _, size := range sizes {
		data := generateLargeSourceMap(size)
		
		b.Run(fmt.Sprintf("Mappings_%d", size), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
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