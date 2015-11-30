package tal

import (
	"bytes"
	"strings"
	"testing"
)

func TestTalesDeepPaths(t *testing.T) {
	type cT struct {
		C map[string]string
		D interface{}
		N interface{}
	}
	type aT struct {
		B map[string]cT
	}
	c := cT{
		C: make(map[string]string),
		D: Default,
		N: None,
	}
	c.C["one"] = "two"
	a := aT{
		B: make(map[string]cT),
	}
	a.B["alpha"] = c

	runTalesTest(t, talesTest{
		struct{ A aT }{A: a},
		`<html><body><h1 tal:content="a/b/alpha/C/one">Default header</h1><h2 tal:content="a/b/alpha/D">Default header 2</h2><h3 tal:content="a/b/alpha/N">Default header 3</h3></body></html>`,
		`<html><body><h1>two</h1><h2>Default header 2</h2><h3></h3></body></html>`,
	})
}

func TestTalesOrPaths(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = None
	vals["b"] = "Hello"
	vals["c"] = "World"

	runTalesTest(t, talesTest{
		vals,
		`<html><body><h1 tal:content="a|b"></h1><h2 tal:content="b|c"></h2><h3 tal:content="a|b|c"></h3></body></html>`,
		`<html><body><h1>Hello</h1><h2>Hello</h2><h3>Hello</h3></body></html>`,
	})
}

func TestTalesRepeatIndex(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = []string{"One", "Two", "Three"}

	runTalesTest(t, talesTest{
		vals,
		`<html><body><p tal:repeat="num a"><b tal:content="repeat/num/index"></b> - <b tal:content="num"></b></p></body></html>`,
		`<html><body><p><b>0</b> - <b>One</b></p><p><b>1</b> - <b>Two</b></p><p><b>2</b> - <b>Three</b></p></body></html>`,
	})
}

func TestTalesAccessStruct(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = struct {
		A string
		B string
		C string
	}{"One", "Two", "Three"}

	runTalesTest(t, talesTest{
		vals,
		`<html><body><p tal:content="a"></p></body></html>`,
		`<html><body><p>?</p></body></html>`,
	})
}

type talesTest struct {
	Context  interface{}
	Template string
	Expected string
}

func runTalesTest(t *testing.T, test talesTest, cfg ...RenderConfig) {
	temp, err := CompileTemplate(strings.NewReader(test.Template))
	if err != nil {
		t.Errorf("Error compiling template: %v\n", err)
		return
	}

	resultBuffer := &bytes.Buffer{}
	err = temp.Render(test.Context, resultBuffer, cfg...)

	if err != nil {
		t.Errorf("Error rendering template: %v\n", err)
		return
	}

	resultStr := resultBuffer.String()

	if resultStr != test.Expected {
		t.Errorf("Expected output: \n%v\nActual output: \n%v\nFrom template: \n%v\nCompiled into: \n%v\n", test.Expected, resultStr, test.Template, temp.String())
		return
	}
}
