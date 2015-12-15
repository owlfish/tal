// Copyright 2015 Colin Stewart.  All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE.txt file.

package tal

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"strings"
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

// attributesList is used to hold attributes before rendering
type attributesList []html.Attribute

// Remove deletes the named attribute from the list
func (a *attributesList) Remove(name string) bool {
	curList := *a
	for i, att := range curList {
		if att.Key == name {
			// Remove this element
			res := append(curList[:i], curList[i+1:]...)
			*a = res
			return true
		}
	}
	return false
}

// Set updates or appends a new attribute with the given values
// The returned bool is true if an update has been done.
func (a *attributesList) Set(name string, value string) bool {
	curList := *a
	for i, att := range curList {
		if att.Key == name {
			// Change this element
			curList[i].Val = value
			return true
		}
	}
	// No existing element with that name - create a new one
	res := append(curList, html.Attribute{Key: name, Val: value})
	*a = res
	return false
}

// Get returns the named attribute, or notFound if not present.
func (a *attributesList) Get(name string) interface{} {
	curList := *a
	for _, att := range curList {
		if att.Key == name {
			return att.Val
		}
	}
	return notFound
}

/*
A templateInstruction provides a render method that is called when the instruction is executed.
*/
type templateInstruction interface {
	// Render the instruction using the provided renderContext
	render(*renderContext) error
}

/*
defineSlot is a template instruction used for metal:define-slot.
*/
type defineSlot struct {
	// name of the slot being defined
	name string
	// endTagOffset holds the distance to the end tag
	endTagOffset int
}

/*
render for a metal:define-slot looks in the current context to see if the slot
has been filled.  If it has, the slot filling is rendered and it skips ahead to
the end of the element scope.

If no slot is defined nothing is done, it continues to render the default
content of the slot.
*/
func (d *defineSlot) render(rc *renderContext) error {
	// Is there a filling for this slot available?
	slotFilling, ok := rc.slots.GetValue(d.name)
	if ok {
		slotFillingTemplate := slotFilling.(*Template)
		// Found a slot filling - substitute it
		err := slotFillingTemplate.renderAsSubtemplate(rc.talesContext, rc.out, rc.slots, rc.config...)
		// Rendered the macro - skip the default content.
		rc.instructionPointer += d.endTagOffset
		return err
	}
	// No slot filling - just output as normal
	return nil
}

// String returns a text description fo the instruction
func (d *defineSlot) String() string {
	return fmt.Sprintf("[Define Slot] %v (end offset %v)", d.name, d.endTagOffset)
}

/*
useMacro is a template instruction used for metal:use-macro.
*/
type useMacro struct {
	// The TALES expression that should resolve into a macro
	expression string
	// originalAttributes are used during TALES expression resolution
	originalAttributes attributesList
	// endTagOffset holds the distance to the end tag
	endTagOffset int
	// filledSlots holds the mapping between the name and template filling any
	// slots in the macro to be used.
	filledSlots map[string]*Template
}

/*
render for a metal:use-macro evaluates the TALES expression and resolves it to
a Template.  All existing slot definitions are saved on a stack (to allow
nesting of macros) and the slots updated with any associated with this useMacro
.  The macro is then rendered, the state of the slots restored and the rest
of the element content skipped.
*/
func (u *useMacro) render(rc *renderContext) error {
	contextValue := rc.talesContext.evaluate(u.expression, u.originalAttributes)
	if contextValue == Default {
		// Continue - use the content of the macro.
		return nil
	}

	if contextValue == nil {
		// No macro - remove the use-macro element
		rc.instructionPointer += u.endTagOffset
		return nil
	}

	mv, ok := contextValue.(*Template)
	if ok {
		// Save current state and add slots
		rc.slots.SaveAll()
		for k, v := range u.filledSlots {
			rc.slots.SetValue(k, v)
		}

		// Render the macro
		err := mv.renderAsSubtemplate(rc.talesContext, rc.out, rc.slots, rc.config...)

		// Now restore the state of the slots before this.
		rc.slots.RestoreAll()

		// Rendered the macro - skip the default content.
		rc.instructionPointer += u.endTagOffset
		return err
	}
	return nil
}

// String returns a text description fo the instruction
func (u *useMacro) String() string {
	return fmt.Sprintf("[Use Macro] %v", u.expression)
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

/*
render for an end tag that contained one or more tal / metal commands.

If a tal:omit-tag command was included in the opening tag, the resolved
value of this is taken out of the context and used to determine whether
it should be rendered.
*/
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

// String returns a text description fo the instruction
func (d *renderEndTag) String() string {
	return fmt.Sprintf("[End Tag] %v (check omit flag: %v)", string(d.tagName), d.checkOmitTagFlag)
}

/*
A defineVariable is used to set local and global variable values.
*/
type defineVariable struct {
	// name is the name of the variable to set
	name string
	// global is true if the definition should be set globally
	global bool
	// expression is the value to set the variable to at runtime
	expression string
	// originalAttributes contains the non-TAL attributes of the original template
	originalAttributes attributesList
}

/*
render for a tal:define command.

If the variable is global it is set, if it is local it is added and the
original value will be restored in the subsequent removeLocalVariable
instruction.
*/
func (d *defineVariable) render(rc *renderContext) error {
	contextValue := rc.talesContext.evaluate(d.expression, d.originalAttributes)
	if d.global {
		rc.talesContext.globalVariables.SetValue(d.name, contextValue)
	} else {
		rc.talesContext.localVariables.AddValue(d.name, contextValue)
	}
	return nil
}

// String returns a text description fo the instruction
func (d *defineVariable) String() string {
	typeOfVar := "local"
	if d.global {
		typeOfVar = "global"
	}

	return fmt.Sprintf("[Define Variable] %v %v to '%v'", typeOfVar, d.name, d.expression)
}

/*
removeLocalVariable removes the most recently defined local variable.
*/
type removeLocalVariable struct {
}

/*
render for removing a local tal:define variable.

The last added local variable is removed, restoring the previous value if
any.
*/
func (d *removeLocalVariable) render(rc *renderContext) error {
	rc.talesContext.localVariables.RemoveValue()
	return nil
}

// String returns a text description fo the instruction
func (d *removeLocalVariable) String() string {
	return "[Remove Local Variable]"
}

/*
renderRepeat is the templateInstruction for repeating blocks of instructions under tal:repeat.
*/
type renderRepeat struct {
	// repeatName is the name used for the local and repeat variable
	repeatName string
	// condition is the TALES expression used for the repeat sequence
	condition string
	// endTagOffset holds the distance to the end tag
	endTagOffset int
	// repeatId holds the unique ID for this repeat, allowing repeatNames to be reused.
	repeatId int
	// originalAttributes contains the non-TAL attributes of the original template
	originalAttributes attributesList
}

/*
render for starting a tal:repeat command.

The TALES expression is evaluated and checked to make sure it is a sequence.
Then a local and repeat variable are established.

If the value is not a sequence type then the instruction jumps to the end tag.
*/
func (d *renderRepeat) render(rc *renderContext) error {
	var contentValue interface{} = nil
	if d.condition != "" {
		contentValue = rc.talesContext.evaluate(d.condition, d.originalAttributes)
	}

	if contentValue == Default {
		// We need to keep the contents intact, but not setup a repeat variable.
		return nil
	}

	if !isValueSequence(contentValue) {
		// Not a sequence, so remove from our flow.
		rc.instructionPointer += d.endTagOffset
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

// String returns a text description fo the instruction
func (d *renderRepeat) String() string {
	return fmt.Sprintf("[Repeat] %v condition '%v' (End Offset %v)", d.repeatName, d.condition, d.endTagOffset)
}

/*
renderEndRepeat is the templateInstruction closing off a tal:repeat.
*/
type renderEndRepeat struct {
	// repeatName is the name used for the local and repeat variable
	repeatName string
	// repeatId holds the unique ID for this repeat, allowing repeatNames to be reused.
	repeatId int
	// repeatStartOffset holds the distance back to the start of the repeat.
	repeatStartOffset int
}

/*
render for ending a loop of a tal:repeat command.

A check is made to confirm that we are already in a loop by looking for a repeat
variable with this name and then verifying the repeatId's match.

The sequence position is advanced.  If this takes it beyond the length of the
sequence, the repeat and local variables are removed.  Otherwise the
instruction pointer is set to the first instruction after the start of the
repeat loop.
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
	rc.instructionPointer += d.repeatStartOffset
	return nil
}

// String returns a text description fo the instruction
func (d *renderEndRepeat) String() string {
	return fmt.Sprintf("[End Repeat] %v (id %v - loop start offset %v)", d.repeatName, d.repeatId, d.repeatStartOffset)
}

// renderData is a template instruction that outputs a slice of bytes
// This is used by text, comments and tags that do not have commands on them.
type renderData struct {
	// data contains the bytes to be written out
	data []byte
}

/*
render for plain text output.
*/
func (d *renderData) render(rc *renderContext) error {
	_, err := rc.out.Write(d.data)
	if err != nil {
		return err
	}
	return nil
}

// String returns a text description fo the instruction
func (d *renderData) String() string {
	dataStr := string(d.data)
	if len(dataStr) > 60 {
		dataStr = dataStr[:60] + "..."
	}
	return fmt.Sprintf("[Output] %v", strings.Replace(dataStr, string('\n'), `\n`, -1))
}

/*
renderCondition is the templateInstruction for tal:condition.
*/
type renderCondition struct {
	// condition holds the TALES expression to be evaluated.
	condition string
	// endTagOffset holds the distance to the end tag
	endTagOffset int
	// originalAttributes contains the non-TAL attributes of the original template
	originalAttributes attributesList
}

/*
render for a tal:condition command.

The condition TALES expression is evaluated and turned into true / false.

If the value is true, nothing else is done.  If false, execution jumps
to after the end tag.
*/
func (d *renderCondition) render(rc *renderContext) error {
	var contentValue interface{} = nil
	if d.condition != "" {
		contentValue = rc.talesContext.evaluate(d.condition, d.originalAttributes)
	}
	if trueOrFalse(contentValue) {
		// Carry on - nothing to do.
		return nil
	}
	rc.instructionPointer += d.endTagOffset

	return nil
}

// String returns a text description fo the instruction
func (d *renderCondition) String() string {
	return fmt.Sprintf("[Condition] '%v' (to offset %v)", d.condition, d.endTagOffset)
}

/*
renderStartTag is the templateInstruction for any tag with commands.
*/
type renderStartTag struct {
	// tagName is the name of the start tag
	tagName []byte
	// contentStructure is true if the content should be treated as structure
	// rather than text
	contentStructure bool
	// contentExpression holds the TALES expression to be evaluated if the
	// content of the tag is to be changed
	contentExpression string
	// originalAttributes holds a copy of the original attributes associated
	// with the start tag
	originalAttributes attributesList
	// attributeExpression holds the list of TALES expressions to be evaluated
	// (i.e. tal:attributes)
	attributeExpression []html.Attribute
	// If replaceCommand is true then the element is replaced entirely
	// (i.e. tal:replace)
	replaceCommand bool
	// endTagOffset holds the relative location of where the corresponding
	// renderEndTag is in the template instructions
	endTagOffset int
	// omitTagExpression is TALES expression associated with tal:omit-tag
	omitTagExpression string
	// voidElement is true if this HTML tag should not have an end tag
	// (e.g. <img>)
	voidElement bool
}

// String returns a text description fo the instruction
func (d *renderStartTag) String() string {
	desc := make(buffer, 0, 240)
	var params []interface{}

	desc.appendString("[Start Tag] %v")
	params = append(params, string(d.tagName))

	if d.contentExpression != "" {
		if d.contentStructure {
			desc.appendString(" structure")
		}
		if d.replaceCommand {
			desc.appendString(" replace with '%v'")
		} else {
			desc.appendString(" content of '%v'")
		}
		params = append(params, d.contentExpression)
	}

	if len(d.attributeExpression) > 0 {
		desc.appendString(" attributes set to %v")
		params = append(params, d.attributeExpression)
	}

	if d.omitTagExpression != "" {
		desc.appendString(" omit tag if '%v'")
		params = append(params, d.omitTagExpression)
	}

	desc.appendString(" (end tag offset %v void element %v)")
	params = append(params, d.endTagOffset)
	params = append(params, d.voidElement)

	return fmt.Sprintf(string(desc), params...)
}

/*
render for a start tag containing commands.

If a tal:omit-tag TALES expression is present, it is evaluated.
If the element is not a void element (e.g. <img>), the resulting
true or false is stored in the render context for the end tag to pickup.

If a contentExpression (from tal:content or tal:replace) is present, it is
evaluated.  If it resolves to Default or it is a tal:content command and
tal:omit-tag is missing or is false, the start tag is rendered.

Start tag rendering checks whether there are any attribute expressions.  If
there are, these are evaluated and the effective attributes updated.

If there is a tal:replace command that did not evaluate to Default, execution
jumps to the end tag.
*/
func (d *renderStartTag) render(rc *renderContext) error {
	// If tal:omit-tag has been used, always ensure that we have called addOmitTagFlag()
	omitTagFlag := false
	if d.omitTagExpression != "" {
		omitTagValue := rc.talesContext.evaluate(d.omitTagExpression, d.originalAttributes)
		omitTagFlag = trueOrFalse(omitTagValue)
		// Add this onto the context
		rc.debug("Omit Tag Flag %v - Omit Tag Value %v - Void %v\n", omitTagFlag, omitTagValue, d.voidElement)
		if !d.voidElement {
			rc.addOmitTagFlag(omitTagFlag)
		}
	}

	var contentValue interface{}
	if d.contentExpression != "" {
		contentValue = rc.talesContext.evaluate(d.contentExpression, d.originalAttributes)
	}

	rc.debug("Start tag content is %v\n", contentValue)

	rc.buffer.reset()
	if contentValue == Default || (!d.replaceCommand && !omitTagFlag) {
		// We are going to write out a start tag, so it's worth evaluating any tal:attribute values at this point.
		var attributes attributesList
		if len(d.attributeExpression) == 0 {
			// No tal:attributes - just use the original values.
			attributes = d.originalAttributes
		} else {
			// Start by taking a copy of the original attributes
			attributes = append(attributes, d.originalAttributes...)
			// Now evaluate each tal:attribute and see what needs to be done.
			for _, talAtt := range d.attributeExpression {
				attValue := rc.talesContext.evaluate(talAtt.Val, d.originalAttributes)
				if attValue == nil {
					// Need to remove this attribute from the list.
					attributes.Remove(talAtt.Key)
				} else if attValue != Default {
					// Over-ride the value
					// If it's a boolean attribute, use the expression to determine what to do.
					_, booleanAtt := htmlBooleanAttributes[talAtt.Key]
					if booleanAtt {
						if trueOrFalse(attValue) {
							// True boolean attributes get the value of their name
							attributes.Set(talAtt.Key, talAtt.Key)
						} else {
							// We remove the attribute
							attributes.Remove(talAtt.Key)
						}
					} else {
						// Normal attribute - just set to the string value.
						attributes.Set(talAtt.Key, fmt.Sprint(attValue))
					}
				}
			}
		}

		rc.buffer.appendString("<")
		rc.buffer.append(d.tagName)
		for _, att := range attributes {
			rc.buffer.appendString(" ")
			rc.buffer.appendString(att.Key)
			rc.buffer.appendString("=\"")
			rc.buffer.appendString(html.EscapeString(att.Val))
			rc.buffer.appendString("\"")
		}
		rc.buffer.appendString(">")
		rc.out.Write(rc.buffer)
	}

	if contentValue == Default || d.contentExpression == "" {
		return nil
	}

	if contentValue != nil {
		if d.contentStructure {
			rc.out.Write([]byte(fmt.Sprint(contentValue)))
		} else {
			rc.out.Write([]byte(html.EscapeString(fmt.Sprint(contentValue))))
		}
	}

	if d.replaceCommand && !d.voidElement {
		rc.debug("Omit Tag is true, jumping to +%v\n", d.endTagOffset)
		rc.instructionPointer += d.endTagOffset
	} else {
		rc.instructionPointer += d.endTagOffset - 1
	}
	return nil
}

/*
renderContext holds the current state of a template rendering.
*/
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
	// omitTagFlags is a stack of bools that is maintained by startTag and endTag to note whether the endTag should be omitted.
	omitTagFlags []bool
	// debug is the logger to use for debug messages
	debug LogFunc
	// original configuration options passed in
	config []RenderConfig
	// slots that have been filled in the template calling this one.
	slots *variableContainer
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

/*
Template holds the compiled version of a TAL template.

Once the Template has been compiled it is immutable and can be used by multiple
goroutines simultaneously.
*/
type Template struct {
	instructions []templateInstruction
	macros       map[string]*Template
}

// newTemplate creates a new empty template.
func newTemplate() *Template {
	return &Template{macros: make(map[string]*Template)}
}

// String creates a full textual description of the compiled template.
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

/*
addRenderInstruction is used to add plain text to the template for output.

If the last instruction in the template is a renderData, it's data is appended
to with the new data.  If not a new renderData instruction is created.
*/
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

// addInstruction appends the given instruction to the template
func (t *Template) addInstruction(instruction templateInstruction) {
	t.instructions = append(t.instructions, instruction)
}

/*
Render a template contents with the given context to the io.Writer.

The Context object should be either a struct or a map with string keys.
Resolution of TAL paths is done in the following order:

	1 - nothing and default: built-in variables
	2 - attrs: access to the original attributes on the element
	3 - repeat: access to the repeat variables
	4 - local variables
	5 - global variables
	6 - User provided context

When looking for a property on local, global and user provided context the
following lookup rules are followed:

	1 - If a map, look for a value with the property name as the key.
	2 - If a struct or pointer to a struct:
		a) Look for an exported field with this name
		b) Look for a Pointer Method with this name and call it
		c) Look for a Value Method with this name and call it

If a value found in either a map or struct field is a function, it will be
called.  Functions and methods are called with no arguments and the first value
returned is used as the resulting value.

A RenderConfig option can be provided to set debug logging.
*/
func (t *Template) Render(context interface{}, out io.Writer, config ...RenderConfig) error {
	rc := &renderContext{
		template:     t,
		out:          out,
		buffer:       make(buffer, 0, 1024),
		talesContext: newTalesContext(context),
		debug:        defaultLogger,
		config:       config,
		slots:        newContainer(),
	}
	for _, c := range config {
		c(t, rc)
	}

	// Put our macros under /macros
	rc.talesContext.globalVariables.SetValue("macros", t)

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

/*
Templates are a TalesValue that provide it's macros as properties.

A Template's own macros are made available to it under the "macros" object.
*/
func (t *Template) TalesValue(name string) interface{} {
	result, ok := t.macros[name]
	if ok {
		return result
	}
	return nil
}

// renderAsSubtemplate is used to render a template into an existing render.
func (t *Template) renderAsSubtemplate(context *tales, out io.Writer, slots *variableContainer, config ...RenderConfig) error {
	rc := &renderContext{
		template:     t,
		out:          out,
		buffer:       make(buffer, 0, 1024),
		talesContext: context,
		debug:        defaultLogger,
		config:       config,
		slots:        slots,
	}
	for _, c := range config {
		c(t, rc)
	}
	// Save global state
	rc.talesContext.globalVariables.SaveAll()
	defer rc.talesContext.globalVariables.RestoreAll()

	// Put our macros under /macros
	// These will be removed by the RestoreAll call
	rc.talesContext.globalVariables.SetValue("macros", t)

	for rc.instructionPointer < len(t.instructions) {
		instruction := t.instructions[rc.instructionPointer]
		rc.debug("Executing renderAsSubtemplate instruction %v\n", instruction)
		err := instruction.render(rc)
		if err != nil {
			return err
		}
		rc.instructionPointer++
	}
	return nil
}

// buffer is a helper type for building byte sequences.
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
