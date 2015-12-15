// Copyright 2015 Colin Stewart.  All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE.txt file.

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
	vals["sharedmacros"] = macroTemplate

	runTalesTest(t, talesTest{
		vals,
		`<html><body><h1 metal:use-macro="sharedmacros/testMacro" tal:content="b">Default B</h1></body></html>`,
		`<html><body><h2>Hello</h2></body></html>`,
	})
}

func TestMetalUseOwnMacro(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"

	runTalesTest(t, talesTest{
		vals,
		`<html><body><h1 metal:define-macro="test" tal:content="b">Default B</h1><div metal:use-macro="macros/test"></div></body></html>`,
		`<html><body><h1>World</h1><h1>World</h1></body></html>`,
	})
}

func TestMetalNilMacro(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"

	runTalesTest(t, talesTest{
		vals,
		`<html><body><h1 metal:define-macro="test" tal:content="b">Default B</h1><div metal:use-macro="nothing">One</div><div metal:use-macro="macros/none">Two</div><div metal:use-macro="default">Three</div></body></html>`,
		`<html><body><h1>World</h1><div>Three</div></body></html>`,
	})
}

func TestMetalDefineAndUseMacroGlobalVar(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><h2 metal:define-macro="testMacro" tal:content="a" tal:define="global news b">Default A</h2></body></html>`))
	vals["sharedmacros"] = macroTemplate

	runTalesTest(t, talesTest{
		vals,
		`<html><body tal:define="global news a"><h1 metal:use-macro="sharedmacros/testMacro" tal:content="b">Default B</h1><i tal:content="news"></i></body></html>`,
		`<html><body><h2>Hello</h2><i>Hello</i></body></html>`,
	})
}

func TestMetalDefineWithSlot(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><p metal:define-macro="testMacro">Hi <b metal:define-slot="name">Default Person</b> there.</p></body></html>`))
	vals["sharedmacros"] = macroTemplate

	runTalesTest(t, talesTest{
		vals,
		`<html><body><div metal:use-macro="sharedmacros/testMacro">Macro content here.</div></body></html>`,
		`<html><body><p>Hi <b>Default Person</b> there.</p></body></html>`,
	})
}

func TestMetalDefineWithFilledSlot(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><p metal:define-macro="testMacro">Hi <b metal:define-slot="name">Default Person</b> there.</p></body></html>`))
	vals["sharedmacros"] = macroTemplate

	runTalesTest(t, talesTest{
		vals,
		`<html><body><div metal:use-macro="sharedmacros/testMacro">Macro <i metal:fill-slot="name">Tester <b>was</b> here.</i> content here.</div></body></html>`,
		`<html><body><p>Hi <i>Tester <b>was</b> here.</i> there.</p></body></html>`,
	})
}

func TestMetalNestedMacros(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<div metal:define-macro="a">Boo <b metal:define-slot="s1">hoo</b></div><div metal:define-macro="b">Hi there <div metal:use-macro="macros/a"><b metal:fill-slot="s1">NO! <span metal:define-slot="s2">Way</span></b></div></div>`))
	vals["sharedmacros"] = macroTemplate

	runTalesTest(t, talesTest{
		vals,
		`<html><body><div metal:use-macro="sharedmacros/b">Macro <i metal:fill-slot="s2">Tester <b>was</b> here.</i> content here.</div></body></html>`,
		`<html><body><div>Hi there <div>Boo <b>NO! <i>Tester <b>was</b> here.</i></b></div></div></body></html>`,
	})
}

func TestMetalDefineOnVoidElement(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><img metal:define-macro="testMacro" href="test image">Hi <b>Default Person</b> there.</body></html>`))
	vals["sharedmacros"] = macroTemplate

	runTalesTest(t, talesTest{
		vals,
		`<html><body><div metal:use-macro="sharedmacros/testMacro">Macro</div>And <img metal:use-macro="sharedmacros/testMacro"> Another</body></html>`,
		`<html><body><img href="test image">And <img href="test image"> Another</body></html>`,
	})
}

func TestMetalSlotsOnVoidElement(t *testing.T) {
	vals := make(map[string]interface{})
	vals["a"] = "Hello"
	vals["b"] = "World"
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><div metal:define-macro="testMacro">Hi <img src="test image" metal:define-slot="one"> There <b>Default Person</b> there.</div></body></html>`))
	vals["sharedmacros"] = macroTemplate

	runTalesTest(t, talesTest{
		vals,
		`<html><body><div metal:use-macro="sharedmacros/testMacro">Macro <i metal:fill-slot="one">I here</i></div> or <div metal:use-macro="sharedmacros/testMacro"> No more <img metal:fill-slot="one" src="alt image"> done. </div></body></html>`,
		`<html><body><div>Hi <i>I here</i> There <b>Default Person</b> there.</div> or <div>Hi <img src="alt image"> There <b>Default Person</b> there.</div></body></html>`,
	})
}
