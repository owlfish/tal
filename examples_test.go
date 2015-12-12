package tal

import (
	"os"
	"strings"
)

func ExampleTemplate_TalesValue() {
	vals := make(map[string]interface{})
	macroTemplate, _ := CompileTemplate(strings.NewReader(`<html><body><h2 metal:define-macro="author">Author Name</h2></body></html>`))
	vals["sharedmacros"] = macroTemplate

	tmpl, _ := CompileTemplate(strings.NewReader(`
		<html><body>
		<p metal:define-macro="boiler">Boiler Plate Message</p>
		<h2 metal:use-macro="sharedmacros/author"></h2>
		<p metal:use-macro="macros/boiler"></p>
		</body></html>`))
	tmpl.Render(vals, os.Stdout)
	/*
		Output:
		<html><body>
		<p>Boiler Plate Message</p>
		<h2>Author Name</h2>
		<p>Boiler Plate Message</p>
		</body></html>
	*/
}

func ExampleTemplate_Render() {
	vals := make(map[string]interface{})
	vals["colours"] = []string{"Red", "Green", "Blue"}
	vals["name"] = "Alice"
	vals["age"] = 21

	tmpl, _ := CompileTemplate(strings.NewReader(`
		<html>
			<body>
				<h1 tal:content="name">Name Here</h1>
				<p tal:content="string: Age: ${age}">Age</p>
				<ul>
					<li tal:repeat="colour colours" tal:content="colour">Colours</li>
				</ul>
			</body>
		</html>`))
	tmpl.Render(vals, os.Stdout)
	/*
		Output:
		<html>
			<body>
				<h1>Alice</h1>
				<p>Age: 21</p>
				<ul>
					<li>Red</li><li>Green</li><li>Blue</li>
				</ul>
			</body>
		</html>
	*/
}
