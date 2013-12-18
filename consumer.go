package sourcemap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/airbrake/sourcemap/base64vlq"
)

type fn func() (fn, error)

type sourceMap struct {
	Version    int           `json:"version"`
	File       string        `json:"file"`
	SourceRoot string        `json:"sourceRoot"`
	Sources    []string      `json:"sources"`
	Names      []interface{} `json:"names"`
	Mappings   string        `json:"mappings"`
}

type Consumer struct {
	baseURL  *url.URL
	smap     *sourceMap
	mappings *mappings
}

func Parse(urlStr string, b []byte) (*Consumer, error) {
	var baseURL *url.URL

	smap := &sourceMap{}
	err := json.Unmarshal(b, smap)
	if err != nil {
		return nil, err
	}

	if smap.Version != 3 {
		return nil, errors.New("sourcemap: only 3rd version is supported")
	}
	if smap.SourceRoot != "" {
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
		if u.IsAbs() {
			baseURL = u
			baseURL.Path = path.Join(baseURL.Path, "_")
		}
	} else {
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
		if u.IsAbs() {
			baseURL = u
		}
	}

	mappings, err := parseMappings(smap.Mappings)
	if err != nil {
		return nil, err
	}
	// Free memory.
	smap.Mappings = ""

	return &Consumer{
		baseURL:  baseURL,
		smap:     smap,
		mappings: mappings,
	}, nil
}

func (c *Consumer) Source(genLine, genCol int) (source, name string, line, col int, ok bool) {
	i := sort.Search(len(c.mappings.values), func(i int) bool {
		m := c.mappings.values[i]
		if m.genLine == genLine {
			return m.genCol >= genCol
		}
		return m.genLine >= genLine
	})

	// Mapping not found.
	if i == len(c.mappings.values) {
		return
	}

	match := c.mappings.values[i]

	// Fuzzy match.
	if match.genCol > genCol {
		match = c.mappings.values[i-1]
	}

	if match.sourcesInd >= 0 {
		source = c.smap.Sources[match.sourcesInd]
		if c.baseURL != nil {
			c.baseURL.Path = path.Join(path.Dir(c.baseURL.Path), source)
			source = c.baseURL.String()
		} else if c.smap.SourceRoot != "" {
			source = path.Join(c.smap.SourceRoot, source)
		}
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

func (c *Consumer) SourceName(genLine, genCol int, genName string) (name string, ok bool) {
	ind := sort.Search(len(c.mappings.values), func(i int) bool {
		m := c.mappings.values[i]
		if m.genLine == genLine {
			return m.genCol >= genCol
		}
		return m.genLine >= genLine
	})

	// Mapping not found.
	if ind == len(c.mappings.values) {
		return
	}

	for i := ind; i >= 0; i-- {
		m := c.mappings.values[i]
		if m.namesInd == -1 {
			continue
		}
		if c.smap.Names[m.namesInd] == "" {

		}
	}

	return
}

type mapping struct {
	genLine    int
	genCol     int
	sourcesInd int
	sourceLine int
	sourceCol  int
	namesInd   int
}

type mappings struct {
	rd  *strings.Reader
	dec *base64vlq.Decoder

	genLine    int
	genCol     int
	sourcesInd int
	sourceLine int
	sourceCol  int
	namesInd   int

	value  *mapping
	values []*mapping
}

func parseMappings(s string) (*mappings, error) {
	rd := strings.NewReader(s)
	m := &mappings{
		rd:  rd,
		dec: base64vlq.NewDecoder(rd),

		genLine:    1,
		sourceLine: 1,
	}
	err := m.parse()
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *mappings) parse() error {
	next := m.parseGenCol
	for {
		if m.value == nil {
			m.value = &mapping{
				genLine:    m.genLine,
				genCol:     0,
				sourcesInd: -1,
				sourceCol:  0,
				namesInd:   -1,
			}
		}

		c, err := m.rd.ReadByte()
		if err == io.EOF {
			m.values = append(m.values, m.value)
			m.value = nil
			return nil
		} else if err != nil {
			return err
		}

		switch c {
		case ',':
			m.values = append(m.values, m.value)
			m.value = nil

			next = m.parseGenCol
		case ';':
			m.values = append(m.values, m.value)
			m.value = nil

			m.genLine++
			m.genCol = 0

			next = m.parseGenCol
		default:
			m.rd.UnreadByte()

			var err error
			next, err = next()
			if err != nil {
				return err
			}
		}
	}
}

func (m *mappings) parseGenCol() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.genCol += n
	m.value.genCol = m.genCol
	return m.parseSourcesInd, nil
}

func (m *mappings) parseSourcesInd() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourcesInd += n
	m.value.sourcesInd = m.sourcesInd
	return m.parseSourceLine, nil
}

func (m *mappings) parseSourceLine() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourceLine += n
	m.value.sourceLine = m.sourceLine
	return m.parseSourceCol, nil
}

func (m *mappings) parseSourceCol() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourceCol += n
	m.value.sourceCol = m.sourceCol
	return m.parseNamesInd, nil
}

func (m *mappings) parseNamesInd() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.namesInd += n
	m.value.namesInd = m.namesInd
	return m.parseGenCol, nil
}
