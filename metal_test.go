package tal

import (
	"strings"
	"testing"
)

func TestMetalDefineMacro(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"

	runTalesTest(t, talesTest{
		vals,
		`<html><body><h1 metal:define-macro="testMacro" tal:content="a"></h1></body></html>`,
		`<html><body><h1>Hello</h1></body></html>`,
	})
}

func TestMetalDefineAndUseMacro(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><h2 metal:define-macro="testMacro" tal:content="a">Default A</h2></body></html>`))
	vals["macros"] = macroTemplate.Macros()

	runTalesTest(t, talesTest{
		vals,
		`<html><body><h1 metal:use-macro="macros/testMacro" tal:content="b">Default B</h1></body></html>`,
		`<html><body><h2>Hello</h2></body></html>`,
	})
}

func TestMetalDefineAndUseMacroGlobalVar(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><h2 metal:define-macro="testMacro" tal:content="a" tal:define="global news b">Default A</h2></body></html>`))
	vals["macros"] = macroTemplate.Macros()

	runTalesTest(t, talesTest{
		vals,
		`<html><body tal:define="global news a"><h1 metal:use-macro="macros/testMacro" tal:content="b">Default B</h1><i tal:content="news"></i></body></html>`,
		`<html><body><h2>Hello</h2><i>Hello</i></body></html>`,
	})
}

func TestMetalDefineWithSlot(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><p metal:define-macro="testMacro">Hi <b metal:define-slot="name">Default Person</b> there.</p></body></html>`))
	vals["macros"] = macroTemplate.Macros()

	runTalesTest(t, talesTest{
		vals,
		`<html><body><div metal:use-macro="macros/testMacro">Macro content here.</div></body></html>`,
		`<html><body><p>Hi <b>Default Person</b> there.</p></body></html>`,
	})
}

func TestMetalDefineWithFilledSlot(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><p metal:define-macro="testMacro">Hi <b metal:define-slot="name">Default Person</b> there.</p></body></html>`))
	vals["macros"] = macroTemplate.Macros()

	runTalesTest(t, talesTest{
		vals,
		`<html><body><div metal:use-macro="macros/testMacro">Macro <i metal:fill-slot="name">Tester <b>was</b> here.</i> content here.</div></body></html>`,
		`<html><body><p>Hi <i>Tester <b>was</b> here.</i> there.</p></body></html>`,
	})
}

func TestMetalNestedMacros(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<div metal:define-macro="a">Boo <b metal:define-slot="s1">hoo</b></div><div metal:define-macro="b">Hi there <div metal:use-macro="macros/a"><b metal:fill-slot="s1">NO! <span metal:define-slot="s2">Way</span></b></div></div>`))
	vals["macros"] = macroTemplate.Macros()

	runTalesTest(t, talesTest{
		vals,
		`<html><body><div metal:use-macro="macros/b">Macro <i metal:fill-slot="s2">Tester <b>was</b> here.</i> content here.</div></body></html>`,
		`<html><body><div>Hi there <div>Boo <b>NO! <i>Tester <b>was</b> here.</i></b></div></div></body></html>`,
	})
}
