package tal

import (
	"bytes"
	"strings"
	"testing"
)

func PassThrough(t *testing.T) {
	runTest(t, talTest{
		struct{}{},
		` <!DOCTYPE html>
		<html>
		<body><h1>Test <b>plan <a>at html</a></b> with an attribute <img src="test.png"></h1><!-- Comment here --></body>
		</html>`,
		` <!DOCTYPE html>
		<html>
		<body><h1>Test <b>plan <a>at html</a></b> with an attribute <img src="test.png"></h1><!-- Comment here --></body>
		</html>`,
	})
}

func TalReplaceSingleTag(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue string
		}{"Replaced Value"},
		`<body><h1>Test <b tal:replace="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test Replaced Value with an attribute <img src="test.png"></h1></body>`,
	})
}

func TalReplaceDefaultValue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{Default},
		`<body><h1>Test <b tal:replace="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test <b class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TalReplaceNoneValue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{None},
		`<body><h1>Test <b tal:replace="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test  with an attribute <img src="test.png"></h1></body>`,
	})
}

func TalContentSimpleValue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{"Simple Value goes here"},
		`<body><h1>Test <b tal:content="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test <b class="test" id="one">Simple Value goes here</b> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TalContentNoneValue(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{None},
		`<body><h1>Test <b tal:content="ContextValue" class="test" id="one">plan <a>at html</a></b> with an attribute <img src="test.png"></h1></body>`,
		`<body><h1>Test <b class="test" id="one"></b> with an attribute <img src="test.png"></h1></body>`,
	})
}

func TalContentDefaultValue(t *testing.T) {
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

type talTest struct {
	Context  interface{}
	Template string
	Expected string
}

func runTest(t *testing.T, test talTest) {
	temp, err := CompileTemplate(strings.NewReader(test.Template))
	if err != nil {
		t.Errorf("Error compiling template: %v\n", err)
		return
	}

	resultBuffer := &bytes.Buffer{}
	err = temp.Render(test.Context, resultBuffer)

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
