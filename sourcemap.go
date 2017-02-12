package sourcemap // import "gopkg.in/sourcemap.v1"

import (
	"io"
	"strings"

	"gopkg.in/sourcemap.v1/base64vlq"
)

type fn func(m *mappings) (fn, error)

type sourceMap struct {
	Version    int           `json:"version"`
	File       string        `json:"file"`
	SourceRoot string        `json:"sourceRoot"`
	Sources    []string      `json:"sources"`
	Names      []interface{} `json:"names"`
	Mappings   string        `json:"mappings"`
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

	values []mapping
	value  *mapping
}

func parseMappings(s string) ([]mapping, error) {
	rd := strings.NewReader(s)
	m := &mappings{
		rd:  rd,
		dec: base64vlq.NewDecoder(rd),

		genLine:    1,
		sourceLine: 1,
	}
	m.pushValue()
	err := m.parse()
	if err != nil {
		return nil, err
	}
	return m.values, nil
}

func (m *mappings) parse() error {
	next := parseGenCol
	for {
		c, err := m.rd.ReadByte()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		switch c {
		case ',':
			m.pushValue()
			next = parseGenCol
		case ';':
			m.pushValue()

			m.genLine++
			m.genCol = 0

			next = parseGenCol
		default:
			err := m.rd.UnreadByte()
			if err != nil {
				return err
			}

			next, err = next(m)
			if err != nil {
				return err
			}
		}
	}
}

func parseGenCol(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.genCol += n
	m.value.genCol = m.genCol
	return parseSourcesInd, nil
}

func parseSourcesInd(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourcesInd += n
	m.value.sourcesInd = m.sourcesInd
	return parseSourceLine, nil
}

func parseSourceLine(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourceLine += n
	m.value.sourceLine = m.sourceLine
	return parseSourceCol, nil
}

func parseSourceCol(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourceCol += n
	m.value.sourceCol = m.sourceCol
	return parseNamesInd, nil
}

func parseNamesInd(m *mappings) (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.namesInd += n
	m.value.namesInd = m.namesInd
	return parseGenCol, nil
}

func (m *mappings) pushValue() {
	m.values = append(m.values, mapping{
		genLine:    m.genLine,
		genCol:     0,
		sourcesInd: -1,
		sourceLine: 0,
		sourceCol:  0,
		namesInd:   -1,
	})
	m.value = &m.values[len(m.values)-1]
}
