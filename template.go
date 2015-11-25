package tal

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"log"
)

type templateInstruction interface {
	render(*renderContext) error
}

type renderEndTag struct {
	tagName          []byte
	checkOmitTagFlag bool
}

func (d *renderEndTag) render(rc *renderContext) error {
	render := true
	log.Printf("Rendering end tag\n")
	if d.checkOmitTagFlag {
		log.Printf("Checking omit tag flag\n")
		render = !rc.getOmitTagFlag()
	}
	if render {
		log.Printf("Rendering OK\n")
		rc.buffer.reset()
		rc.buffer.appendString("</")
		rc.buffer.append(d.tagName)
		rc.buffer.appendString(">")
		rc.out.Write(rc.buffer)
	} else {
		log.Printf("Rendering of end tag suppressed.\n")
	}
	return nil
}

func (d *renderEndTag) String() string {
	return fmt.Sprintf("</%v> omit flag test: %v", string(d.tagName), d.checkOmitTagFlag)
}

type renderData struct {
	data []byte
}

func (d *renderData) render(rc *renderContext) error {
	_, err := rc.out.Write(d.data)
	if err != nil {
		return err
	}
	return nil
}

func (d *renderData) String() string {
	dataStr := string(d.data)
	if len(dataStr) > 60 {
		dataStr = dataStr[:60]
	}
	return dataStr
}

type renderCondition struct {
	condition   string
	endTagIndex int
}

func (d *renderCondition) render(rc *renderContext) error {
	var contentValue interface{} = None
	if d.condition != "" {
		contentValue = rc.talesContext.evaluate(d.condition)
	}
	if trueOrFalse(contentValue) {
		// Carry on - nothing to do.
		return nil
	}
	rc.instructionPointer = d.endTagIndex

	return nil
}

type renderStartTag struct {
	tagName              []byte
	contentStructure     bool
	contentExpression    string
	originalAttributes   []html.Attribute
	attributesExpression string
	replaceCommand       bool
	endTagIndex          int
	omitTagExpression    string
	voidElement          bool
}

func (d *renderStartTag) String() string {
	return fmt.Sprintf("<%v> start tag - contentStructure %v - contentExpression %v - omitTagExpression %v", string(d.tagName), d.contentStructure, d.contentExpression, d.omitTagExpression)
}

func (d *renderStartTag) render(rc *renderContext) error {
	// TODO - Evaluate content
	// TODO - Evaluate attributes

	// If tal:omit-tag has been used, always ensure that we have called addOmitTagFlag()
	omitTagFlag := false
	if d.omitTagExpression != "" {
		omitTagValue := rc.talesContext.evaluate(d.omitTagExpression)
		omitTagFlag = trueOrFalse(omitTagValue)
		// Add this onto the context
		log.Printf("Omit Tag Flag %v - Omit Tag Value %v - Void %v\n", omitTagFlag, omitTagValue, d.voidElement)
		if !d.voidElement {
			rc.addOmitTagFlag(omitTagFlag)
		}
	}

	var contentValue interface{}
	if d.contentExpression != "" {
		contentValue = rc.talesContext.evaluate(d.contentExpression)
	}

	log.Printf("Start tag content is %v\n", contentValue)

	rc.buffer.reset()
	if contentValue == Default || (!d.replaceCommand && !omitTagFlag) {
		rc.buffer.appendString("<")
		rc.buffer.append(d.tagName)
		for _, att := range d.originalAttributes {
			rc.buffer.appendString(" ")
			rc.buffer.appendString(att.Key)
			rc.buffer.appendString("=\"")
			rc.buffer.appendString(html.EscapeString(att.Val))
			rc.buffer.appendString("\"")
		}
		rc.buffer.appendString(">")
		rc.out.Write(rc.buffer)
	}

	if contentValue == Default || contentValue == nil {
		return nil
	}

	if contentValue != None {
		rc.out.Write([]byte(fmt.Sprint(contentValue)))
	}

	if d.replaceCommand {
		log.Printf("Omit Tag is true, jumping to %v\n", d.endTagIndex)
		rc.instructionPointer = d.endTagIndex
	} else {
		rc.instructionPointer = d.endTagIndex - 1
	}
	return nil
}

type renderContext struct {
	template           *Template
	position           int
	context            interface{}
	out                io.Writer
	buffer             buffer
	talesContext       *tales
	instructionPointer int
	omitTagFlags       []bool
}

/*
getOmitTagFlag returns the last omit tag flag state on the render context stack.
The flag is true if the end tag should be omitted from output, false otherwise.
*/
func (rc *renderContext) getOmitTagFlag() bool {
	// We should always have a flag available, but don't panic if we don't
	flagsLength := len(rc.omitTagFlags)
	if flagsLength == 0 {
		log.Printf("Unexpected render error - getOmitTagFlag called, but no flags available!\n")
		return false
	}
	result := rc.omitTagFlags[flagsLength-1]
	rc.omitTagFlags = rc.omitTagFlags[:flagsLength-1]
	return result
}

/*
addOmitTagFlag puts the result of an omit-tag calculation onto the render context stack.
This will be picked up by the renderEndTag for tags carrying the tal:omit-tag command.
*/
func (rc *renderContext) addOmitTagFlag(flag bool) {
	rc.omitTagFlags = append(rc.omitTagFlags, flag)
}

type Template struct {
	instructions []templateInstruction
}

func (t *Template) String() string {
	buf := make(buffer, 0, 100)
	for index, instr := range t.instructions {
		buf.appendStringF("%v: %v\n", index, instr)
	}
	buf = append(buf, []byte("Start Test")...)
	buf.appendString("Append test")
	buf = append(buf, []byte("Test")...)
	return string(buf)
}

func (t *Template) addRenderInstruction(data []byte) {
	// If there are already instructions, see if they can be merged
	if len(t.instructions) > 0 {
		lastInstructionPos := len(t.instructions) - 1
		renderDataInstr, ok := t.instructions[lastInstructionPos].(*renderData)
		if ok {
			renderDataInstr.data = append(renderDataInstr.data, data...)
			return
		}
		// Last instruction was not a renderData
	}
	// If we've made it here, we need to create and add a new instruction.
	t.instructions = append(t.instructions, &renderData{data})
}

func (t *Template) addInstruction(instruction templateInstruction) {
	t.instructions = append(t.instructions, instruction)
}

func (t *Template) Render(context interface{}, out io.Writer) error {
	rc := &renderContext{
		template:     t,
		position:     0,
		context:      context,
		out:          out,
		buffer:       make(buffer, 0, 100),
		talesContext: newTalesContext(context),
	}
	for rc.instructionPointer < len(t.instructions) {
		instruction := t.instructions[rc.instructionPointer]
		err := instruction.render(rc)
		if err != nil {
			return err
		}
		rc.instructionPointer++
	}
	return nil
}

type buffer []byte

func (b *buffer) append(newb []byte) {
	var newBuff buffer = append(*b, newb...)
	*b = newBuff
}

func (b *buffer) appendString(newstr string) {
	var newBuff buffer = append(*b, []byte(newstr)...)
	*b = newBuff
}

func (b *buffer) appendStringF(newstr string, params ...interface{}) {
	var newBuff buffer = append(*b, []byte(fmt.Sprintf(newstr, params...))...)
	*b = newBuff
}

func (b *buffer) reset() {
	var curBuff buffer = *b
	newBuff := curBuff[:0]
	*b = newBuff
}
