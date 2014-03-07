package sourcemap_test

import (
	"io/ioutil"
	"net/http"
	"sync"
	"testing"

	. "launchpad.net/gocheck"

	"github.com/airbrake/sourcemap"
)

const (
	jqSourceMapURL = "http://code.jquery.com/jquery-2.0.3.min.map"
)

var (
	jqSourceMapOnce sync.Once
	_jqSourceMap    []byte
)

func jqSourceMap() []byte {
	jqSourceMapOnce.Do(func() {
		resp, err := http.Get(jqSourceMapURL)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		_jqSourceMap = b
	})

	return _jqSourceMap
}

func Test(t *testing.T) { TestingT(t) }

type SourceMapTest struct{}

var _ = Suite(&SourceMapTest{})

func (t *SourceMapTest) TestSourceMap(c *C) {
	smap, err := sourcemap.Parse("", sourceMapJSON)
	if err != nil {
		panic(err)
	}

	table := []struct {
		genLine int
		genCol  int
		source  string
		name    string
		line    int
		col     int
	}{
		{1, 1, "/the/root/one.js", "", 1, 1},
		{1, 5, "/the/root/one.js", "", 1, 5},
		{1, 9, "/the/root/one.js", "", 1, 11},
		{1, 18, "/the/root/one.js", "bar", 1, 21},
		{1, 21, "/the/root/one.js", "", 2, 3},
		{1, 28, "/the/root/one.js", "baz", 2, 10},
		{1, 32, "/the/root/one.js", "bar", 2, 14},

		{2, 1, "/the/root/two.js", "", 1, 1},
		{2, 5, "/the/root/two.js", "", 1, 5},
		{2, 9, "/the/root/two.js", "", 1, 11},
		{2, 18, "/the/root/two.js", "n", 1, 21},
		{2, 21, "/the/root/two.js", "", 2, 3},
		{2, 28, "/the/root/two.js", "n", 2, 10},

		// Fuzzy match.
		{1, 20, "/the/root/one.js", "bar", 1, 21},
		{1, 30, "/the/root/one.js", "baz", 2, 10},
		{2, 12, "/the/root/two.js", "", 1, 11},
	}
	for _, row := range table {
		source, name, line, col, ok := smap.Source(row.genLine, row.genCol)
		c.Assert(ok, Equals, true, Commentf("%#v", row))
		c.Assert(source, Equals, row.source, Commentf("%#v", row))
		c.Assert(name, Equals, row.name, Commentf("%#v", row))
		c.Assert(line, Equals, row.line, Commentf("%#v", row))
		c.Assert(col, Equals, row.col, Commentf("%#v", row))
	}

	_, _, _, _, ok := smap.Source(3, 0)
	c.Assert(ok, Equals, false)
}

func (t *SourceMapTest) TestJQuerySourceMap(c *C) {
	smap, err := sourcemap.Parse(jqSourceMapURL, jqSourceMap())
	c.Assert(err, IsNil)

	table := []struct {
		genLine int
		genCol  int
		source  string
		name    string
		line    int
		col     int
	}{
		{5, 6789, "http://code.jquery.com/jquery-2.0.3.js", "apply", 4360, 27},
		{5, 10006, "http://code.jquery.com/jquery-2.0.3.js", "apply", 4676, 8},
		{4, 553, "http://code.jquery.com/jquery-2.0.3.js", "ready", 93, 9},
	}
	for _, row := range table {
		source, name, line, col, ok := smap.Source(row.genLine, row.genCol)
		c.Assert(ok, Equals, true, Commentf("%#v", row))
		c.Assert(source, Equals, row.source, Commentf("%#v", row))
		c.Assert(name, Equals, row.name, Commentf("%#v", row))
		c.Assert(line, Equals, row.line, Commentf("%#v", row))
		c.Assert(col, Equals, row.col, Commentf("%#v", row))
	}
}

func (t *SourceMapTest) BenchmarkParse(c *C) {
	c.StopTimer()
	b := jqSourceMap()
	c.StartTimer()

	for i := 0; i < c.N; i++ {
		_, err := sourcemap.Parse(jqSourceMapURL, b)
		if err != nil {
			panic(err)
		}
	}
}

func (t *SourceMapTest) BenchmarkSource(c *C) {
	c.StopTimer()
	b := jqSourceMap()
	smap, err := sourcemap.Parse(jqSourceMapURL, b)
	if err != nil {
		panic(err)
	}
	c.StartTimer()

	for i := 0; i < c.N; i++ {
		for j := 0; j < 10; j++ {
			smap.Source(j, 100*j)
		}
	}

}

// This is a test mapping which maps functions from two different files
// (one.js and two.js) to a minified generated source.
//
// Here is one.js:
//
//     ONE.foo = function (bar) {
//       return baz(bar);
//     };
//
// Here is two.js:
//
//     TWO.inc = function (n) {
//       return n + 1;
//     };
//
// And here is the generated code (min.js):
//
//     ONE.foo=function(a){return baz(a);};
//     TWO.inc=function(a){return a+1;};

var genCode = []byte(`exports.testGeneratedCode = "ONE.foo=function(a){return baz(a);};
TWO.inc=function(a){return a+1;};`)

var sourceMapJSON = []byte(`{
  "version": 3,
  "file": "min.js",
  "names": ["bar", "baz", "n"],
  "sources": ["one.js", "two.js"],
  "sourceRoot": "/the/root",
  "mappings": "CAAC,IAAI,IAAM,SAAUA,GAClB,OAAOC,IAAID;CCDb,IAAI,IAAM,SAAUE,GAClB,OAAOA"
}`)
