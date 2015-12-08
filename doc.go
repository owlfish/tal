// Copyright 2015 Colin Stewart.  All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE.txt file.

/*
Package tal implements the TAL template language for generating HTML5 output.

TAL templates are HTML5 documents containing additional attributes that, using the TALES expression language, change the structure and content of the resulting document.

Here is a basic example (without error handling) that prints "<html><h1>Raising Steam</h1> by Author: Terry Pratchett"

	type Book struct {
		Title  string
		Author string
	}

	book := Book{"Raising Steam", "Terry Pratchett"}
	tmpl, _ := tal.CompileTemplate(strings.NewReader(`<html><h1 tal:content="title">Title goes here</h1> by <b tal:replace="string: Author: $Author"></b>`))
	tmpl.Render(book, os.Stdout)

The tal package also supports METAL macros as described later on.

TAL Commands

The tal package supports all TAL commands defined by Zope (see http://docs.zope.org/zope2/zope2book/AppendixC.html).


Define

tal:define sets a local or global variable:

	tal:define="[local | global] name expression [; define-expression...]

Description: Sets the value of "name" to "expression".  By default the name will be applicable in the "local" scope, which consists of this tag, and all other tags nested inside this tag.  If the "global" keyword is used then this name will keep its value for the rest of the document.

Example:

	<div tal:define="global title book/theTitle; local chapterTitle book/chapter/theTitle">

Condition

tal:condition makes output of an element conditional:

	tal:condition="expression"

Description:  If the expression evaluates to true then this tag and all its children will be output.  If the expression evaluates to false then this tag and all its children will not be included in the output.

Example:

	<h1 tal:condition="user/firstLogin">Welcome to this page!</h1>

Repeat

tal:repeat replicates an element a number of times:

	tal:repeat="name expression"

Description:  Evaluates "expression", and if it is a sequence, repeats this tag and all children once for each item in the sequence.  The "name" will be set to the value of the item in the current iteration, and is also the name of the repeat variable.  The repeat variable is accessible using the TAL path: repeat/name and has the following properties:

    index 		- Iteration number starting from zero
    number 		- Iteration number starting from one
    even 		- True if this is an even iteration
    odd 		- True if this is an odd iteration
    start 		- True if this is the first item in the sequence
    end 		- True if this is the last item in the sequence
    length 		- The length of the sequence
    letter 		- The lower case letter for this iteration, starting at "a"
    Letter		- Upper case version of letter
    roman 		- Iteration number in Roman numerals, starting at i
    Roman	 	- Upper case version of roman

Note that letterUpper and romanUpper are used instead of the standard TAL Letter and Roman.  The "first" and "last" properties are not supported.

Example:

	<table>
		<tr tal:repeat="fruit basket">
			<td tal:content="repeat/fruit/number"></td>
			<td tal:content="fruit/name"></td>
		</tr>
	</table>

Content

tal:content replaces the content of an element:

	tal:content="[text | structure] expression"

Description:  Replaces the contents of the tag with the value of "expression".  By default, and if the "text" keyword is present, then the value of the expression will be escaped as required (i.e. characters "&<> will be escaped).  If the "structure" keyword is present then the value will be output with no escaping performed.

Example:

	<h1 tal:content="user/firstName"></h1>

Replace

tal:replace replaces the whole element:

	tal:replace="[text | structure] expression"

Description: Behaves identically to tal:content, except that the tag is removed from the output (as if tal:omit-tag had been used).

Example:

	<h1>Welcome <b tal:replace="user/firstName"></b></h1>

Attributes

tal:attributes sets or removes attributes on the element:

	tal:attributes="name expression[;attributes-expression]"

Description:  Evaluates each "expression" and replaces the tag's attribute "name".  If the expression evaluates to nothing then the attribute is removed from the tag.  If the expression evaluates to default then the original tag's attribute is kept.  If the "expression" requires a semi-colon then it must be escaped by using ";;".

Example:

	<a tal:attributes="href user/homepage;title user/fullname">Your Homepage</a>

Omit Tag

tal:omit-tag removes the start and end tags:

	tal:omit-tag="expression"

Description: Removes the tag (leaving the tags content) if the expression evaluates to true.  If expression is empty then it is taken as true.

Example:

	<p><b tal:omit-tag="not:user/firstVisit">Welcome</b> to this page!</h1>

TALES Expressions

The expressions used in TAL are called TALES expressions.  The simplest TALES expression is a path which references a value, e.g. page/body references the body property of the page object.  Objects are passed as the first argument of the Render method on a compiled template and must be either a struct, pointer to a struct or a map with strings as keys.

The tal package does not support the python: and nocall: expression types.

Path

path: provides access to properties on objects.

	Syntax: [path:]string[|TALES Expression]

Description: A path, optionally starting with the modifier 'path:', references a property of an object.  The '/' delimiter is used to end the name of an object and start of the property name.  Properties themselves may be objects that in turn have properties.  The '|' ("or") character is used to find an alternative value to a path if the first path evaluates does not exist.

Example:

	<p tal:content="book/chapter/title | string:Untitled"></p>

There are several built in variables that can be used in paths:

    nothing	- acts as nil in Go
    default	- keeps the existing value of the node (tag content or attribute value)
    repeat	- access to repeat variables (see tal:repeat)
    attrs	- a dictionary of original attributes of the current tag

Path Variables

Path variables allows for indirection in paths.

	Syntax: [path:]object/?attribute

Description:  The "attribute" is evaluated as a local or global variable, and it's value is used as the attribute to lookup on the object.  Useful for accessing the contents of a map within a loop:

Example:

	<div tal:content="myMap/?loopValue"/>

Exists

exists: Tests whether a path exists.

	Syntax: exists:path

Description: Returns true if the path exists, false otherwise.  Particularly useful for eliminating entire subtrees if a particular object is not available.

Example:

	<div tal:condition="exists:book">...</div>

Not

not: Returns the inverse boolean value of a path.

	Syntax: not:tales-path

Description: Returns the inverse of the tales-path.  If the path returns true, not:path will return false.

Example:

	<p tal:condition="not: user/firstLogin">Welcome to the site!</p>

String

string: Basic string substitution

	Syntax: string:text

Description:  Evaluates to a string with value text while substituting variables with the form ${pathName} and $pathName

Example:

	<b tal:content="string:Welcome ${user/name}!"></b>

METAL Macro Language

METAL is a macro language commonly used with TAL & TALES.  METAL allows part of a template to be used as a macro in later parts of the template, or shared across templates.

Macros defined in a template can be retrieved from a compiled template and added to the context by calling the Macros() method.

Define Macro

metal:define-macro marks an element and it's subtree as being a reusable macro.

	Syntax: metal:define-macro="name"

Description: Defines a new macro that can be reference later as "name".

Example:

	<div metal:define-macro="footer">Copyright <span tal:content="page/lastModified">2004</span></div>

Use Macro

metal:use-macro expands a macro into the template.

	Syntax: metal:use-macro="expression"

Description: Evaluates "expression" and uses this as a macro.

Example:

	<div metal:use-macro="macros/footer"></div>

Define Slot

metal:define-slot creates a customisation point within a macro.

	Syntax: metal:define-slot="name"

Description: Defines a customisation point in a macro with the given name.

Example:

	<div metal:define-macro="footer">
		<b>Standard disclaimer for the site.</b>
		<i metal:define-slot="Contact">Contact admin@site.com</i>
	</div>

Fill Slot

metal:fill-slot overwrites the contents of a slot when using a macro.

Syntax: metal:fill-slot="name"

Description: Replaces the content of a slot with this element.

Example:

	<div metal:use-macro="macros/footer">
		<i metal:fill-slot="Contact">Contact someone else</i>
	</div>

*/
package tal
