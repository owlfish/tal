package tal

import (
	"bytes"
	//"fmt"
	"html/template"
	"strings"
	"testing"
)

var performanceTemplate string = `<html>
<head>
  <title></title>
      
  <meta http-equiv="content-type"
 content="text/html; charset=ISO-8859-1">
   
  <meta name="author" content="Colin Stewart">
</head>
<body>
 
<h1 tal:content="title">Performance Template</h1>
<div tal:repeat="things myList">
<h2 tal:content="string: $things/title itteration">Itteration title</h2>
<p tal:repeat="content things/content">
	<b tal:content="content/colour">Colour</b>
	<ul>
	  <li tal:repeat="anum content/num" tal:content="anum">All numbers</li>
	</ul>
</p>
</div>
 
</body>
</html>
`

var goPerformanceTemplate string = `<html>
<head>
  <title></title>
      
  <meta http-equiv="content-type"
 content="text/html; charset=ISO-8859-1">
   
  <meta name="author" content="Colin Stewart">
</head>
<body>
 
<h1>{{.title}}</h1>
{{range .myList}}<div>
<h2>{{.title}} itteration</h2>
{{range .content}}<p>
	<b>{{.colour}}</b>
	<ul>
	  {{range .num}}<li>{{.}}</li>{{end}}
	</ul>
</p>{{end}}
</div>{{end}}
</body>
</html>
`

// 3 X 7 X 8 = 168 itterations per template expansion.
var thirdLevelList = []string{"One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight"}

var secondLevelList = []map[string]interface{}{
	{"colour": "red", "num": thirdLevelList},
	{"colour": "orange", "num": thirdLevelList},
	{"colour": "yellow", "num": thirdLevelList},
	{"colour": "green", "num": thirdLevelList},
	{"colour": "blue", "num": thirdLevelList},
	{"colour": "indigo", "num": thirdLevelList},
	{"colour": "violet", "num": thirdLevelList},
}

var firstLevelList = []map[string]interface{}{
	{"title": "First", "content": secondLevelList},
	{"title": "Second", "content": secondLevelList},
	{"title": "Third", "content": secondLevelList},
}

//[{"colour": "red", "num": thirdLevelList}, {"colour": "orange", "num": thirdLevelList}, {"colour": "yellow", "num": thirdLevelList}, {"colour": "green", "num": thirdLevelList}, {"colour": "blue", "num": thirdLevelList}, {"colour": "indigo", "num": thirdLevelList}, {"colour": "violet", "num": thirdLevelList}]
//firstLevelList = [{"title": "First", "content": secondLevelList}, {"title": "Second", "content": secondLevelList}, {"title": "Third", "content": secondLevelList}]

//context = simpleTALES.Context()
//context.addGlobal ("title", "Performance testing!")
//context.addGlobal ("myList", firstLevelList )

func BenchmarkDeeplyNestedRepeat(b *testing.B) {
	temp, err := CompileTemplate(strings.NewReader(performanceTemplate))
	if err != nil {
		b.Errorf("Error compiling template: %v\n", err)
		return
	}

	context := make(map[string]interface{})
	context["title"] = "Performance testing!"
	context["myList"] = firstLevelList

	resultBuffer := &bytes.Buffer{}
	err = temp.Render(context, resultBuffer)

	//fmt.Printf(resultBuffer.String())

	for i := 0; i < b.N; i++ {
		//log.Printf("Template: %v\n", temp)
		resultBuffer.Reset()
		err = temp.Render(context, resultBuffer)

		if err != nil {
			b.Errorf("Error rendering template: %v\n", err)
			return
		}
	}
}

// BenchmarkDeeplyNestedRepeatGoTemplate provides a comparison point against which TAL can be measured
func BenchmarkDeeplyNestedRepeatGoTemplate(b *testing.B) {
	temp, err := template.New("name").Parse(goPerformanceTemplate)
	if err != nil {
		b.Errorf("Error compiling template: %v\n", err)
		return
	}

	context := make(map[string]interface{})
	context["title"] = "Performance testing!"
	context["myList"] = firstLevelList

	resultBuffer := &bytes.Buffer{}
	temp.Execute(resultBuffer, context)

	//fmt.Printf(resultBuffer.String())

	for i := 0; i < b.N; i++ {
		//log.Printf("Template: %v\n", temp)
		resultBuffer.Reset()
		err = temp.Execute(resultBuffer, context)

		if err != nil {
			b.Errorf("Error rendering template: %v\n", err)
			return
		}
	}
}

func BenchmarkDeeplyNestedRepeatCompile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		//log.Printf("Template: %v\n", temp)
		_, err := CompileTemplate(strings.NewReader(performanceTemplate))
		if err != nil {
			b.Errorf("Error compiling template: %v\n", err)
			return
		}
	}
}
