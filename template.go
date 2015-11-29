package tal

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
)

/*
A RenderConfig function is one that can be passed as an option to Render.
*/
type RenderConfig func(t *Template, rc *renderContext)

/*
RenderDebugLogging uses the given LogFunc for debug output when rendering the template.

To use the standard log library pass RenderDebugLogging(log.Printf) to the Render method.
*/
func RenderDebugLogging(logger LogFunc) RenderConfig {
	return func(t *Template, rc *renderContext) {
		rc.talesContext.debug = logger
		rc.debug = logger
	}
}

/*
A templateInstruction provides a render method that can output part of a template given the current renderContext.
*/
type templateInstruction interface {
	render(*renderContext) error
}

/*
A renderEndTag is used to render the close tag of an HTML element that contains one or more TAL commands.
*/
type renderEndTag struct {
	// tagName is the name of the tag that should be closed.
	tagName []byte
	// checkOmitTagFlag is true if the tag had a tal:omit-tag command on it.
	// If the flag is true then the context is checked to see whether the end tag should be omitted.
	checkOmitTagFlag bool
}

func (d *renderEndTag) render(rc *renderContext) error {
	render := true
	if d.checkOmitTagFlag {
		rc.debug("Checking omit tag flag\n")
		render = !rc.getOmitTagFlag()
	}
	if render {
		rc.debug("End Tag will be rendered\n")
		rc.buffer.reset()
		rc.buffer.appendString("</")
		rc.buffer.append(d.tagName)
		rc.buffer.appendString(">")
		rc.out.Write(rc.buffer)
	} else {
		rc.debug("Rendering of end tag suppressed.\n")
	}
	return nil
}

func (d *renderEndTag) String() string {
	return fmt.Sprintf("</%v> omit flag test: %v", string(d.tagName), d.checkOmitTagFlag)
}

/*
renderRepeat is the templateInstruction for repeating blocks of instructions under tal:repeat.
*/
type renderRepeat struct {
	repeatName  string
	condition   string
	endTagIndex int
	repeatId    int
}

/*
TODO: Write render for renderRepeat
*/
func (d *renderRepeat) render(rc *renderContext) error {
	// Check to see whether we are already doing a repeat sequence for this tag.
	repeatVar, ok := rc.talesContext.repeatVariables.GetValue(d.repeatName)
	if ok {
		repeatVar := repeatVar.(*repeatVariable)
		if repeatVar.repeatId == d.repeatId {
			// We have a match - we are already repeating and so nothing to do but continue
			return nil
		}
	}

	var contentValue interface{} = None
	if d.condition != "" {
		contentValue = rc.talesContext.evaluate(d.condition)
	}

	if contentValue == Default {
		// We need to keep the contents intact, but not setup a repeat variable.
		return nil
	}

	if !isValueSequence(contentValue) {
		// Not a sequence, so remove from our flow.
		rc.instructionPointer = d.endTagIndex
		return nil
	}
	// We have a sequenece, need to iterate over it.
	// Setup the repeat value
	newRepeatVar := newRepeatVariable(d.repeatId, contentValue)
	rc.talesContext.repeatVariables.AddValue(d.repeatName, newRepeatVar)
	// Create and set the local variable to the first element
	rc.talesContext.localVariables.AddValue(d.repeatName, newRepeatVar.indexedValue())

	return nil
}

func (d *renderRepeat) String() string {
	return fmt.Sprintf("Repeat %v (condition %v) to index %v", d.repeatName, d.condition, d.endTagIndex)
}

/*
renderEndRepeat is the templateInstruction closing off a tal:repeat.
*/
type renderEndRepeat struct {
	repeatName       string
	repeatId         int
	repeatStartIndex int
}

/*
TODO: Write render for renderEndRepeat
*/
func (d *renderEndRepeat) render(rc *renderContext) error {
	// Check to see whether we are doing a repeat sequence.
	candidate, ok := rc.talesContext.repeatVariables.GetValue(d.repeatName)

	if !ok {
		// We are not repeating, just continue.
		return nil
	}
	repeatVar := candidate.(*repeatVariable)
	if repeatVar.repeatId != d.repeatId {
		// The repeat variable is from a different sequence - just continue.
		return nil
	}

	// We are doing a genuine repeat - need to advance and see if we should continue.
	repeatVar.sequencePosition++
	if repeatVar.sequencePosition == repeatVar.sequenceLength {
		// This is the end of the repeat - remove the repeat and local variables.
		rc.talesContext.repeatVariables.RemoveValue()
		rc.talesContext.localVariables.RemoveValue()
		return nil
	}
	// Update the value of the local variable.
	rc.talesContext.localVariables.SetValue(d.repeatName, repeatVar.indexedValue())

	// Finally loop back around the start tag.
	rc.instructionPointer = d.repeatStartIndex
	return nil
}

func (d *renderEndRepeat) String() string {
	return fmt.Sprintf("END Repeat %v (id %v) start index %v", d.repeatName, d.repeatId, d.repeatStartIndex)
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
		rc.debug("Omit Tag Flag %v - Omit Tag Value %v - Void %v\n", omitTagFlag, omitTagValue, d.voidElement)
		if !d.voidElement {
			rc.addOmitTagFlag(omitTagFlag)
		}
	}

	var contentValue interface{}
	if d.contentExpression != "" {
		contentValue = rc.talesContext.evaluate(d.contentExpression)
	}

	rc.debug("Start tag content is %v\n", contentValue)

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
		rc.debug("Omit Tag is true, jumping to %v\n", d.endTagIndex)
		rc.instructionPointer = d.endTagIndex
	} else {
		rc.instructionPointer = d.endTagIndex - 1
	}
	return nil
}

type renderContext struct {
	// template holders the reference to the template being executed.
	template *Template
	// out is where the rendered template should be written to.
	out io.Writer
	// buffer is a temporary buffer available for individual instructions to use.
	buffer buffer
	// talesContext holds the local, global and repeat variables and the context supplied to Render.
	talesContext *tales
	// instructionPointer holds the index of the instruction in the template being executed.
	instructionPointer int
	// omitTagFlags is a stack of bools that is maintained by startTag and endTag to note whether the endTag should be ommitted.
	omitTagFlags []bool
	// debug is the logger to use for debug messages
	debug LogFunc
}

/*
getOmitTagFlag returns the last omit tag flag state on the render context stack.
The flag is true if the end tag should be omitted from output, false otherwise.
*/
func (rc *renderContext) getOmitTagFlag() bool {
	// We should always have a flag available, but don't panic if we don't
	flagsLength := len(rc.omitTagFlags)
	if flagsLength == 0 {
		rc.debug("Unexpected render error - getOmitTagFlag called, but no flags available!\n")
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

func (t *Template) Render(context interface{}, out io.Writer, config ...RenderConfig) error {
	rc := &renderContext{
		template:     t,
		out:          out,
		buffer:       make(buffer, 0, 100),
		talesContext: newTalesContext(context),
		debug:        defaultLogger,
	}
	for _, c := range config {
		c(t, rc)
	}
	for rc.instructionPointer < len(t.instructions) {
		instruction := t.instructions[rc.instructionPointer]
		rc.debug("Executing instruction %v\n", instruction)
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
