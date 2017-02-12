package sourcemap

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
)

type Consumer struct {
	sourceRootURL *url.URL
	smap          *sourceMap
	mappings      []mapping
}

func Parse(mapURL string, b []byte) (*Consumer, error) {
	smap := new(sourceMap)
	err := json.Unmarshal(b, smap)
	if err != nil {
		return nil, err
	}

	if smap.Version != 3 {
		return nil, errors.New("sourcemap: only 3rd version is supported")
	}

	var sourceRootURL *url.URL
	if smap.SourceRoot != "" {
		u, err := url.Parse(smap.SourceRoot)
		if err != nil {
			return nil, err
		}
		if u.IsAbs() {
			sourceRootURL = u
		}
	} else if mapURL != "" {
		u, err := url.Parse(mapURL)
		if err != nil {
			return nil, err
		}
		if u.IsAbs() {
			u.Path = path.Dir(u.Path)
			sourceRootURL = u
		}
	}

	mappings, err := parseMappings(smap.Mappings)
	if err != nil {
		return nil, err
	}
	// Free memory.
	smap.Mappings = ""

	return &Consumer{
		sourceRootURL: sourceRootURL,
		smap:          smap,
		mappings:      mappings,
	}, nil
}

func (c *Consumer) File() string {
	return c.smap.File
}

func (c *Consumer) Source(genLine, genCol int) (source, name string, line, col int, ok bool) {
	i := sort.Search(len(c.mappings), func(i int) bool {
		m := &c.mappings[i]
		if m.genLine == genLine {
			return m.genCol >= genCol
		}
		return m.genLine >= genLine
	})

	// Mapping not found.
	if i == len(c.mappings) {
		return
	}

	match := &c.mappings[i]

	// Fuzzy match.
	if match.genCol > genCol && i > 0 {
		match = &c.mappings[i-1]
	}

	if match.sourcesInd >= 0 {
		source = c.absSource(c.smap.Sources[match.sourcesInd])
	}
	if match.namesInd >= 0 {
		iv := c.smap.Names[match.namesInd]
		switch v := iv.(type) {
		case string:
			name = v
		case float64:
			name = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			name = fmt.Sprint(iv)
		}
	}
	line = match.sourceLine
	col = match.sourceCol
	ok = true
	return
}

func (c *Consumer) absSource(source string) string {
	if path.IsAbs(source) {
		return source
	}

	if u, err := url.Parse(source); err == nil && u.IsAbs() {
		return source
	}

	if c.sourceRootURL != nil {
		u := *c.sourceRootURL
		u.Path = path.Join(c.sourceRootURL.Path, source)
		return u.String()
	}

	if c.smap.SourceRoot != "" {
		return path.Join(c.smap.SourceRoot, source)
	}

	return source
}

func (c *Consumer) SourceName(genLine, genCol int, genName string) (name string, ok bool) {
	ind := sort.Search(len(c.mappings), func(i int) bool {
		m := c.mappings[i]
		if m.genLine == genLine {
			return m.genCol >= genCol
		}
		return m.genLine >= genLine
	})

	// Mapping not found.
	if ind == len(c.mappings) {
		return "", false
	}

	for i := ind; i >= 0; i-- {
		m := c.mappings[i]
		if m.namesInd == -1 {
			continue
		}
		if c.smap.Names[m.namesInd] == "" {

		}
	}

	return
}
