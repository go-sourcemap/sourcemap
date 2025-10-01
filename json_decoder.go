//go:build !jsonv2
// +build !jsonv2

package sourcemap

import "encoding/json"

// unmarshalJSON is the JSON unmarshaling function
// This version uses the standard encoding/json package
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}