package sourcemap

import (
	"reflect"
	"testing"
)

func TestParseMappings(t *testing.T) {
	t.Parallel()
	cases := map[string][]mapping{
		";;;;;;kBAEe,YAAY,CAC1B,C;;AAHD": {
			{genLine: 7, genColumn: 18, sourceLine: 3, sourceColumn: 15, namesInd: -1},
			{genLine: 7, genColumn: 30, sourceLine: 3, sourceColumn: 27, namesInd: -1},
			{genLine: 7, genColumn: 31, sourceLine: 4, sourceColumn: 1, namesInd: -1},
			{genLine: 7, genColumn: 32, sourceLine: 4, sourceColumn: 1, namesInd: -1},
			{genLine: 9, genColumn: 0, sourceLine: 1, sourceColumn: 0, namesInd: -1},
		},
	}
	for k, c := range cases {
		k, c := k, c
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			v, err := parseMappings(k)
			if err != nil {
				t.Fatalf("got error %s", err)
			}
			if !reflect.DeepEqual(v, c) {
				t.Fatalf("expected %v got %v", c, v)
			}
		})
	}
}
