package tal

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

var debug RenderConfig = RenderDebugLogging(log.Printf)

func TestSplitDefineArguments(t *testing.T) {
	testStr := "local one;global two;local three;;four;global five"
	res := splitTalArguments(testStr)
	expected := []string{"local one", "global two", "local three;four", "global five"}
	if len(res) != len(expected) {
		t.Errorf("String split resulted in %v not %v\n", res, expected)
		return
	}
	for i, part := range expected {
		if res[i] != part {
			t.Errorf("String split resulted in %v not %v\n", res, expected)
		}
	}
}

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

func TestTalReplaceTextKeyword(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{"<b>Some bold & text</b>"},
		`<body><p tal:replace="text ContextValue">plan</p></body>`,
		`<body>&lt;b&gt;Some bold &amp; text&lt;/b&gt;</body>`,
	})
}

func TestTalReplaceTextKeywordNoExpression(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
			Text         interface{}
		}{"<b>Some bold & text</b>",
			"Test Text"},
		`<body><p tal:replace="text">plan</p></body>`,
		`<body>Test Text</body>`,
	})
}

func TestTalReplaceStructureKeyword(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{"<b>Some bold &amp; text</b>"},
		`<body><p tal:replace="structure ContextValue">plan</p></body>`,
		`<body><b>Some bold &amp; text</b></body>`,
	})
}

func TestTalReplaceStructureKeywordNoExpression(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
			Structure    interface{}
		}{"<b>Some bold & text</b>",
			"Test Text"},
		`<body><p tal:replace="structure">plan</p></body>`,
		`<body>Test Text</body>`,
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

func TestTalContentText(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{"<b>Some bold & text</b>"},
		`<body><p tal:content="ContextValue">plan</p></body>`,
		`<body><p>&lt;b&gt;Some bold &amp; text&lt;/b&gt;</p></body>`,
	})
}

func TestTalContentTextKeyword(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{"<b>Some bold & text</b>"},
		`<body><p tal:content="text ContextValue">plan</p></body>`,
		`<body><p>&lt;b&gt;Some bold &amp; text&lt;/b&gt;</p></body>`,
	})
}

func TestTalContentTextKeywordNoExpression(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
			Text         interface{}
		}{"<b>Some bold & text</b>",
			"Test Text"},
		`<body><p tal:content="text">plan</p></body>`,
		`<body><p>Test Text</p></body>`,
	})
}

func TestTalContentStructureKeyword(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
		}{"<b>Some bold &amp; text</b>"},
		`<body><p tal:content="structure ContextValue">plan</p></body>`,
		`<body><p><b>Some bold &amp; text</b></p></body>`,
	})
}

func TestTalContentStructureKeywordNoExpression(t *testing.T) {
	runTest(t, talTest{
		struct {
			ContextValue interface{}
			Structure    interface{}
		}{"<b>Some bold & text</b>",
			"Test Text"},
		`<body><p tal:content="structure">plan</p></body>`,
		`<body><p>Test Text</p></body>`,
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

func TestTalDefineLocalNoKeyword(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
		}{"One"},
		`<body><p tal:define="avar Value" tal:content="avar"></p><b tal:content="avar"></b></body>`,
		`<body><p>One</p><b></b></body>`,
	})
}

func TestTalDefineLocalKeyword(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
		}{"One"},
		`<body><p tal:define="local avar Value" tal:content="avar"></p><b tal:content="avar"></b></body>`,
		`<body><p>One</p><b></b></body>`,
	})
}

func TestTalDefineGlobalKeyword(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
		}{"One"},
		`<body><p tal:define="global avar Value" tal:content="avar"></p><b tal:content="avar"></b></body>`,
		`<body><p>One</p><b>One</b></body>`,
	})
}

func TestTalDefineLocalNested(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
			V2    interface{}
			V3    interface{}
		}{"One",
			"Two",
			"Three"},
		`<body><p tal:define="local avar Value"><h1 tal:replace="avar"></h1><b tal:define="avar V2"><i tal:replace="avar"></i><span tal:define="avar V3"><i tal:replace="avar"></i></span><i tal:replace="avar"></i></b><i tal:replace="avar"></i></p></body>`,
		`<body><p>One<b>Two<span>Three</span>Two</b>One</p></body>`,
	})
}

func TestTalDefineGlobalAndLocalKeyword(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
			V2    interface{}
		}{"One", "Two"},
		`<body><p tal:define="global avar Value;local bvar V2"><h1 tal:content="avar"></h1><h2 tal:content="bvar"></h2></p><b tal:content="avar"></b><i tal:content="bvar"></i></body>`,
		`<body><p><h1>One</h1><h2>Two</h2></p><b>One</b><i></i></body>`,
	})
}

func TestTalAttributesNew(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
			V2    interface{}
		}{"One", "Two"},
		`<body><h1 tal:attributes="href V2">Test</h1></body>`,
		`<body><h1 href="Two">Test</h1></body>`,
	})
}

func TestTalAttributesAdditional(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
			V2    interface{}
		}{"One", "Two"},
		`<body><h1 class="class-one" id="#1" tal:attributes="href V2">Test</h1></body>`,
		`<body><h1 class="class-one" id="#1" href="Two">Test</h1></body>`,
	})
}

func TestTalAttributesRemove(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
			V2    interface{}
		}{"One", "Two"},
		`<body><h1 class="class-one" id="#1" tal:attributes="class None">Test</h1></body>`,
		`<body><h1 id="#1">Test</h1></body>`,
	})
}

func TestTalAttributesDefault(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
			V2    interface{}
			V3    interface{}
		}{"One", "Two", Default},
		`<body><h1 class="class-one" id="#1" tal:attributes="class V3">Test</h1></body>`,
		`<body><h1 class="class-one" id="#1">Test</h1></body>`,
	})
}

func TestTalAttributesMany(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
			V2    interface{}
			V3    interface{}
		}{"One", "Two", Default},
		`<body><h1 class="class-one" id="#1" tal:attributes="class V3;id v2;href Value">Test</h1></body>`,
		`<body><h1 class="class-one" id="Two" href="One">Test</h1></body>`,
	})
}

func TestTalAttributesWithContent(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
			V2    interface{}
			V3    interface{}
		}{"One", "Two", Default},
		`<body><h1 class="class-one" id="#1" tal:attributes="class V3;id v2;href Value" tal:content="Value">Test</h1></body>`,
		`<body><h1 class="class-one" id="Two" href="One">One</h1></body>`,
	})
}

func TestTalAttributesWithRepeat(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value []interface{}
		}{
			[]interface{}{"One", "Two", Default, "Three", None, "Four"},
		},
		`<body><ul><li tal:repeat="num Value" tal:attributes="id num" id="default-num">Test</li></ul></body>`,
		`<body><ul><li id="One">Test</li><li id="Two">Test</li><li id="default-num">Test</li><li id="Three">Test</li><li>Test</li><li id="Four">Test</li></ul></body>`,
	})
}

func TestTalAttributesBoolean(t *testing.T) {
	runTest(t, talTest{
		struct {
			Value interface{}
			V2    interface{}
			V3    interface{}
			V4    interface{}
		}{"One", "Two", true, false},
		`<body><h1 tal:attributes="checked V3;default V4" tal:content="Value">Test</h1></body>`,
		`<body><h1 checked="checked">One</h1></body>`,
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
	//log.Printf("Template: %v\n", temp)

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
