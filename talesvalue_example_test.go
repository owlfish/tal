// Copyright 2015 Colin Stewart.  All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE.txt file.

package tal

import (
	"os"
	"strings"
)

type Person struct {
	name string
}

func (p *Person) TalesValue(property string) interface{} {
	switch property {
	case "Name":
		return p.name
	case "upper":
		return strings.ToUpper(p.name)
	case "lower":
		return strings.ToLower(p.name)
	}
	return nil
}

func ExampleTalesValue() {

	vals := make(map[string]interface{})
	vals["person"] = &Person{"Alice"}

	tmpl, _ := CompileTemplate(strings.NewReader(`<b tal:content="person/Name"></b> and <b tal:content="person/upper"></b> and <b tal:content="person/lower"></b>`))
	tmpl.Render(vals, os.Stdout)
	// Output: <b>Alice</b> and <b>ALICE</b> and <b>alice</b>
}
