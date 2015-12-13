# tal - Go Implementation of Template Attribute Language

[TAL](http://docs.zope.org/zope2/zope2book/AppendixC.html) is a template language created by the Zope project and implemented in a number of [different languages](https://en.wikipedia.org/wiki/Template_Attribute_Language).  The language is very compact, consisting of just 7 commands with a further 4 METAL commands available for macros.  This makes learning and understanding TAL more straightforward than many other templating languages.

The tal library provides an implementation of TAL (and TALES and METAL) for Go.  Performance of tal is similar to the standard library's html/template (benchmarks show faster execution but more allocations).

A simple example of how to use tal:

```
package main

import "github.com/owlfish/tal"
import "os"
import "strings"

func main() {
	template := `
		<html>
			<h1 tal:content="Title">Title Here</h1>
			<div tal:repeat="book Library">
				<h2 tal:content="book/Title">Book Title</h2>
				<b tal:content="book/Author">Author</b>
				<p tal:condition="book/Classification">Classification <b tal:replace="book/Classification">Book Type</b></p>
			</div>
		</html>
	`

	type Book struct {
		Title          string
		Author         string
		Classification string
	}

	books := []Book{{"Raising Steam", "Terry Pratchett", "Fiction"}, {Title: "My Life", Author: "Anon"}}

	context := make(map[string]interface{})
	context["Library"] = books
	context["Title"] = "Library"

	tmpl, _ := tal.CompileTemplate(strings.NewReader(template))
	tmpl.Render(context, os.Stdout)
}
```
The output from the above is:
```
		<html>
			<h1>Library</h1>
			<div>
				<h2>Raising Steam</h2>
				<b>Terry Pratchett</b>
				<p>Classification Fiction</p>
			</div>
			<div>
				<h2>My Life</h2>
				<b>Anon</b>
			</div>
		</html>
```

To get started with the latest version of tal use `go get github.com/owlfish/tal`.  Documentation is at ...

## Status

Development is now complete with a full implementation of TAL, TALES and METAL.  The documentation, examples, tests and benchmarks are at a reasonable level of completeness (>94% test coverage).

No API changes are planned at this point, although behaviour may change if required to address defects.  When I wrote a Python implementation of TAL, [SimpleTAL](http://owlfish.com/software/simpleTAL/), I found and fixed a great number of issues over time.  The Go implementation uses a lot of the learning from SimpleTAL, so should hopefully mature more quickly.
