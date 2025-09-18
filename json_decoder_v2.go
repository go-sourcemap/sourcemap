//go:build jsonv2
// +build jsonv2

package sourcemap

import "encoding/json/v2"

// unmarshalJSON is the JSON unmarshaling function
// This version uses the experimental json/v2 package for better performance
// Build with: GOEXPERIMENT=jsonv2 go build -tags=jsonv2 ./...
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}