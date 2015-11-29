package tal

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

var debug RenderConfig = RenderDebugLogging(log.Printf)

func TestPassThrough(t *testing.T) {
	runTest(t, talTest{
		struct{}{},
		` <!DOCTYPE html>
		<html>
		<body><h1>Test &lt; &amp; &gt; <b>plan <a>at html</a></b> with an attribute <img src="test.png"></h1><!-- Comment here --></body>
		</html>`,
		` <!DOCTYPE html>
		<html>
		<body><h1>Test &lt; &amp; &gt; <b>plan <a>at html</a></b> with an attribute <img src="test.png"></h1><!-- Comment here --></body>
		</html>`,
	})
}

func TestTalReplaceSingleTag(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue string
		}{"Replaced Value"},
		`<body><h1>Test <b tal:replace="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test Replaced Value with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalReplaceDefaultValue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{Default},
		`<body><h1>Test <b tal:replace="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test <b class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalReplaceNoneValue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{None},
		`<body><h1>Test <b tal:replace="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test  with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalContentSimpleValue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{"Simple Value goes here"},
		`<body><h1>Test <b tal:content="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test <b class="test" id="one">Simple Value goes here</b> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalContentNoneValue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{None},
		`<body><h1>Test <b tal:content="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test <b class="test" id="one"></b> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalContentDefaultValue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{Default},
		`<body><h1>Test <b tal:content="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test <b class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalConditionFalse(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{false},
		`<body><h1>Test <b tal:condition="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test  with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalConditionTrue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{true},
		`<body><h1>Test <b tal:condition="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test <b class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalOmitTagFalse(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{false},
		`<body><h1>Test <b tal:omit-tag="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test <b class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalOmitTagTrue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{true},
		`<body><h1>Test <b tal:omit-tag="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test plan <a>at html</a> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TestTalRepeatNoneSequence(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{false},
		`<body><h1>Test</h1> <ul> <li tal:repeat="vals ContextValue" class="line-item">Value <b tal:content="vals">Vals go here</b> done.</li></ul></body>`,
		`<body><h1>Test</h1> <ul> </ul></body>`,
	})
}

func TestTalRepeatDefault(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
			Vals         string
		}{
			Default,
			"Default vals",
		},
		`<body><h1>Test</h1> <ul> <li tal:repeat="vals ContextValue" class="line-item">Value <b tal:content="vals">Vals go here</b> done.</li></ul></body>`,
		`<body><h1>Test</h1> <ul> <li class="line-item">Value <b>Default vals</b> done.</li></ul></body>`,
	})
}

func TestTalRepeatOneEntry(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue []string
			Vals         string
		}{
			[]string{"One value"},
			"Default vals",
		},
		`<body><h1>Test</h1> <ul> <li tal:repeat="vals ContextValue" class="line-item">Value <b tal:content="vals">Vals go here</b> done.</li></ul><p tal:content="vals"></p></body>`,
		`<body><h1>Test</h1> <ul> <li class="line-item">Value <b>One value</b> done.</li></ul><p>Default vals</p></body>`,
	})
}

func TestTalRepeatTwoEntries(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue []string
			Vals         string
		}{
			[]string{"One value", "Two values"},
			"Default vals",
		},
		`<body><h1>Test</h1> <ul> <li tal:repeat="vals ContextValue" class="line-item">Value <b tal:content="vals">Vals go here</b> done.</li></ul></body>`,
		`<body><h1>Test</h1> <ul> <li class="line-item">Value <b>One value</b> done.</li><li class="line-item">Value <b>Two values</b> done.</li></ul></body>`,
	})
}

type talTest struct {
	Context  interface{}
	Template string
	Expected string
}

func runTest(t *testing.T, test talTest, cfg ...RenderConfig) {
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
