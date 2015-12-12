package tal

import (
	"bytes"
	"golang.org/x/net/html"
	"io"
	"sort"
	"strings"
)

/*
A LogFunc is a function that can be used for logging.  log.Printf is a LogFunc.
*/
type LogFunc func(fmt string, args ...interface{})

// defaultLogger does nothing - just returns
func defaultLogger(fmt string, args ...interface{}) {
}

/*
An endActionFunc is executed when a tal element's end tag is seen.

Multiple endActionFunc's can be associated with a given end tag, allowing
multiple actions to be composed.
*/
type endActionFunc func()

/*
startActionFunc is executed when a tal command is seen in a start tag.

Multiple startActionFunc's may be associated with a given start tag, allowing
multiple tal commands to be composed.
*/
type startActionFunc func(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError

/*
tagInfo records the tag and slice of end actions to be executed when the end
tag is seen.
*/
type tagInfo struct {
	tag        []byte
	popActions []endActionFunc
}

/*
compileState holds the state of a particular compilation while underway.
*/
type compileState struct {
	/*
		tagStack is a LIFO slice of end tags and actions.
		These are compared against as actual end tags are seen.
	*/
	tagStack []tagInfo
	// template holds the pointer to the template being constructed.
	template *Template
	// tokenizer holds a reference to the HTML tokenizer being used.
	tokenizer *html.Tokenizer
	/*
		talStartTag holds the start tag instruction that is being constructed.
		Each startActionFunc and endActionFunc has an opportunity to modify the
		values of the talStartTag.
	*/
	talStartTag *renderStartTag
	/*
		talEndTag holds the end tag instruction that is being constructed.
		Each startActionFunc and endActionFunc has an opportunity to modify the
		values of the talEndTag.
	*/
	talEndTag *renderEndTag
	// nextId is used to create unique IDs for repeat actions.
	nextId int
	// currentMacro holds the last metal:use-macro command seen.
	currentMacro *useMacro
}

/*
addTag adds the given tag name to the stack of seen tags.
*/
func (state *compileState) addTag(tag []byte) {
	state.tagStack = append(state.tagStack, tagInfo{tag: tag})
}

/*
appendAction associates an action to be taken when the tag is pop'd from the stack
*/
func (state *compileState) appendAction(action endActionFunc) {
	state.tagStack[len(state.tagStack)-1].popActions = append(state.tagStack[len(state.tagStack)-1].popActions, action)
}

/*
insertAction places an endActionFunc at the start of the list of end actions to be taken
*/
func (state *compileState) insertAction(action endActionFunc) {
	// Extend the length of the list of actions
	state.tagStack[len(state.tagStack)-1].popActions = append(state.tagStack[len(state.tagStack)-1].popActions, action)
	// Now shuffle everything up one, overwriting the new entry we just made
	copy(state.tagStack[len(state.tagStack)-1].popActions[1:], state.tagStack[len(state.tagStack)-1].popActions[:])
	// Finally, update the first entry
	state.tagStack[len(state.tagStack)-1].popActions[0] = action
}

/*
popTag removes the tag from the stack and executes any end actions.

popTag keeps removing tags until the one given is found, or no tags remain.
*/
func (state *compileState) popTag(tag []byte) error {
	if len(state.tagStack) > 0 {
		candidate := state.tagStack[len(state.tagStack)-1]
		state.tagStack = state.tagStack[:len(state.tagStack)-1]
		// Run any actions
		for _, act := range candidate.popActions {
			act()
		}
		if bytes.Equal(candidate.tag, tag) {
			return nil
		}
		//state.Printf("Mis-Matched tags %s and %s\n", candidate.tag, tag)
	}
	return newCompileError(ErrUnexpectedCloseTag, state.tokenizer.Raw(), state.tokenizer.Buffered())
}

/*
error returns a CompileError with the context of where it happened.
*/
func (state *compileState) error(errorType int) *CompileError {
	return newCompileError(errorType, state.tokenizer.Raw(), state.tokenizer.Buffered())
}

// talAttributes are a slice of html.Attribute with helper methods for sorting.
type talAttributes []html.Attribute

/*
talCommandProperties holds the command priorities and startActionFuncs.

All tal and metal attribute commands are sorted by the Priority before being
handled.  Each startActionFunc is executed in turn.
*/
var talCommandProperties = map[string]struct {
	Priority    int
	StartAction startActionFunc
}{
	"metal:define-macro": {0, metalDefineMacroStart},
	"metal:use-macro":    {1, metalUseMacroStart},
	"metal:define-slot":  {2, metalDefineSlotStart},
	"metal:fill-slot":    {3, metalFillSlotStart},
	"tal:define":         {4, talDefineStart},
	"tal:condition":      {5, talConditionStart},
	"tal:repeat":         {6, talRepeatStart},
	"tal:content":        {7, talContentStart},
	"tal:replace":        {8, talReplaceStart},
	"tal:attributes":     {9, talAttributesStart},
	"tal:omit-tag":       {10, talOmitTagStart},
}

// talCommandPriority returns the priority of a command
func talCommandPriority(command string) int {
	properties, ok := talCommandProperties[command]
	if !ok {
		return 100
	}
	return properties.Priority
}

func (s talAttributes) Len() int {
	return len(s)
}
func (s talAttributes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s talAttributes) Less(i, j int) bool {
	iPriority := talCommandPriority(s[i].Key)
	jPriority := talCommandPriority(s[j].Key)

	return iPriority < jPriority
}

/*
splitTalArguments returns a slice of ";" separated commands.

Semi-colons can be escaped using ";;"
This is used for both tal:define and tal:attributes.
*/
func splitTalArguments(value string) []string {
	parts := strings.Split(value, ";")
	var results []string
	var candidate string
	escapedSepFound := false
	for _, part := range parts {
		if len(part) == 0 {
			// We have a double escape - append a semi-colon
			candidate = candidate + ";"
			escapedSepFound = true
		} else {
			// Is this part of an escape char?
			if escapedSepFound {
				candidate += part
			} else {
				if len(candidate) > 0 {
					// Clear up the old candidate
					results = append(results, candidate)
				}
				candidate = part
			}
			escapedSepFound = false
		}
	}
	if len(candidate) > 0 {
		// Clear up the old candidate
		results = append(results, candidate)
	}
	return results
}

/*
metalDefineSlotStart is used for metal:define-slot.

A new defineSlot template instruction is created and metalDefineSlotEndAction
is registered as an end action to calculate the offset to the end tag.
*/
func metalDefineSlotStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	ds := &defineSlot{name: talValue}
	state.template.addInstruction(ds)

	state.appendAction(metalDefineSlotEndAction(state, ds))
	return nil
}

/*
metalDefineSlotEndAction is used for the end tag of a metal:define-slot.

It calculates the offset from the defineSlot instruction to the end tag.
*/
func metalDefineSlotEndAction(state *compileState, ds *defineSlot) endActionFunc {
	startPoint := len(state.template.instructions)

	return func() {
		ds.endTagOffset = len(state.template.instructions) - startPoint
	}
}

/*
metalFillSlotStart is used for metal:fill-slot.

A check is made to ensures that the fill-slot is nested inside a use-macro.

metalFillSlotEnd is called to register an end action to do the work.
*/
func metalFillSlotStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	// Check that we are inside a macro
	if state.currentMacro == nil {
		return state.error(ErrSlotOutsideMacro)
	}
	// Defer to the end of the slot definition to register it
	state.appendAction(metalFillSlotEnd(talValue, state))
	return nil
}

/*
metalFillSlotEnd is used for the end tag of a metal:fill-slot.

It creates a new Template for the instructions contained within the fill-slot
element and registers this with the given name in the current use-macro.
*/
func metalFillSlotEnd(name string, state *compileState) endActionFunc {
	startPoint := len(state.template.instructions)
	return func() {
		slotTemplate := newTemplate()
		slotTemplate.instructions = state.template.instructions[startPoint:]
		slotTemplate.macros = state.template.macros
		state.currentMacro.filledSlots[name] = slotTemplate
	}
}

/*
metalUseMacroStart is used for metal:use-macro.

A useMacro template instruction is added to the template.
metalUseMacroEndAction then completes the rest of the work.
*/
func metalUseMacroStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	// Create a useMacro template instruction
	um := &useMacro{expression: talValue, originalAttributes: originalAttributes, filledSlots: make(map[string]*Template)}
	state.template.addInstruction(um)
	// Add the end tag index when we know it.
	state.appendAction(metalUseMacroEndAction(state, um))
	return nil
}

/*
metalUseMacroEndAction is used for the end tag of a metal:use-macro.

It records the location of the newly created useMacro instruction, saves the
existing currentMacro (in case of nesting) and sets currentMacro.

The returned endActionFunc calculates the offset from the start of the useMacro
and restores the previous value of currentMacro.
*/
func metalUseMacroEndAction(state *compileState, um *useMacro) endActionFunc {
	umLocation := len(state.template.instructions)
	// Record the current macro and keep the old one for restore
	// This allows metal:fill-slot to find the current macro.
	previousMacro := state.currentMacro
	state.currentMacro = um
	return func() {
		um.endTagOffset = len(state.template.instructions) - umLocation
		// Restore the previous macro - required for nested macros.
		state.currentMacro = previousMacro
	}
}

/*
metalDefineMacroStart is used for metal:define-macro.

All work is deferred to metalDefineMacroEndAction.
*/
func metalDefineMacroStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	// Do all the work at the end.
	state.appendAction(metalDefineMacroEndAction(state.template, talValue, len(state.template.instructions)))
	return nil
}

/*
metalDefineMacroEndAction is used for the end tag of a metal:define-macro.

A new template is created covering all instructions from the start of
define-macro to the end of the element.  This template is then added under the
given name into the map of macros.
*/
func metalDefineMacroEndAction(t *Template, name string, startInstructionIndex int) endActionFunc {
	return func() {
		// Create a new template for the macro
		// Contains all instructions from the start to the end last instruction created.
		macroTemplate := newTemplate()
		macroTemplate.instructions = t.instructions[startInstructionIndex:]
		macroTemplate.macros = t.macros
		t.macros[name] = macroTemplate
	}
}

/*
talAttributesStart is used for tal:attributes.

All arguments are split and the resulting name / value pairs are appended to
the startTag's attribute expression list.  No endAction is used.
*/
func talAttributesStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	definitionList := splitTalArguments(talValue)
	for _, definition := range definitionList {
		actualDef := strings.Split(definition, " ")
		if len(actualDef) == 2 {
			state.talStartTag.attributeExpression = append(state.talStartTag.attributeExpression, html.Attribute{Key: actualDef[0], Val: actualDef[1]})
		} else {
			return state.error(ErrExpressionMissing)
		}
	}
	return nil
}

/*
talDefineStart is used for tal:define.

All arguments are split and local / global scope is determined.
defineVariable template instructions are created for each variable defined
and each local variable definition results in a endAction from
getTalDefineEndAction being registered.
*/
func talDefineStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	definitionList := splitTalArguments(talValue)
	for _, definition := range definitionList {
		if strings.HasPrefix(definition, "local ") && len(definition) > 6 {
			actualDef := strings.Split(definition[6:], " ")
			if len(actualDef) == 2 {
				state.template.addInstruction(&defineVariable{name: actualDef[0], global: false, expression: actualDef[1], originalAttributes: originalAttributes})
				// Local variables need popping when the end tag is seen.
				state.appendAction(getTalDefineEndAction(state.template))
			} else {
				return state.error(ErrExpressionMissing)
			}
		} else if strings.HasPrefix(definition, "global ") && len(definition) > 7 {
			actualDef := strings.Split(definition[7:], " ")
			if len(actualDef) == 2 {
				state.template.addInstruction(&defineVariable{name: actualDef[0], global: true, expression: actualDef[1], originalAttributes: originalAttributes})
			} else {
				return state.error(ErrExpressionMissing)
			}
		} else {
			// Treat as a local variable defintion.
			actualDef := strings.Split(definition, " ")
			if len(actualDef) == 2 {
				state.template.addInstruction(&defineVariable{name: actualDef[0], global: false, expression: actualDef[1], originalAttributes: originalAttributes})
				// Local variables need popping when the end tag is seen.
				state.appendAction(getTalDefineEndAction(state.template))
			} else {
				return state.error(ErrExpressionMissing)
			}
		}
	}
	return nil
}

/*
getTalDefineEndAction is used for the end tag of a tal:define.

This is only used for local variables.  A new template instruction of
removeLocalVariable is added to the template instruction list.
*/
func getTalDefineEndAction(t *Template) endActionFunc {
	return func() {
		// Add a local variable remove instruction.
		t.addInstruction(&removeLocalVariable{})
	}
}

/*
talReplaceStart is used for tal:replace.

text / structure is determined and a values on the existing
talStartTag are changed as required.  No endAction is used.
*/
func talReplaceStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	if len(talValue) == 0 {
		return state.error(ErrExpressionMissing)
	}
	state.talStartTag.replaceCommand = true

	// If we start with "text " and have an expression after that, remove the prefix
	if strings.HasPrefix(talValue, "text ") && len(talValue) > 5 {
		state.talStartTag.contentExpression = talValue[5:]
	} else if strings.HasPrefix(talValue, "structure ") && len(talValue) > 10 {
		state.talStartTag.contentExpression = talValue[10:]
		state.talStartTag.contentStructure = true
	} else {
		state.talStartTag.contentExpression = talValue
	}

	return nil
}

/*
talContentStart is used for tal:content.

text / structure is determined and a values on the existing
talStartTag are changed as required.  No endAction is used.
*/
func talContentStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	if len(talValue) == 0 {
		return state.error(ErrExpressionMissing)
	}
	state.talStartTag.replaceCommand = false

	// If we start with "text " and have an expression after that, remove the prefix
	if strings.HasPrefix(talValue, "text ") && len(talValue) > 5 {
		state.talStartTag.contentExpression = talValue[5:]
	} else if strings.HasPrefix(talValue, "structure ") && len(talValue) > 10 {
		state.talStartTag.contentExpression = talValue[10:]
		state.talStartTag.contentStructure = true
	} else {
		state.talStartTag.contentExpression = talValue
	}
	return nil
}

/*
talConditionStart is used for tal:condition.

A new renderCondition template instruction is created.
An end action from getTalConditionEndAction is registered.
*/
func talConditionStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	if len(talValue) == 0 {
		return state.error(ErrExpressionMissing)
	}
	condition := renderCondition{condition: talValue, originalAttributes: originalAttributes}
	state.template.addInstruction(&condition)
	state.appendAction(getTalConditionEndAction(state.template, &condition))
	return nil
}

/*
getTalConditionEndAction is used for the end tag of a tal:condition.

The returned endActionFunc calculates the offset from the condition to the
end tag.
*/
func getTalConditionEndAction(t *Template, condition *renderCondition) endActionFunc {
	startLocation := len(t.instructions)
	return func() {
		// This action is executed once the end tag is seen
		// condition and state is captured inside this closure
		condition.endTagOffset = len(t.instructions) - startLocation
	}
}

/*
talRepeatStart is used for tal:repeat.

A new renderRepeat template instruction is created.
An end action from getTalRepeatEndAction is registered.
*/
func talRepeatStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	parts := strings.Split(talValue, " ")
	if len(parts) != 2 {
		return state.error(ErrExpressionMalformed)
	}
	repeat := renderRepeat{repeatName: parts[0], condition: parts[1], repeatId: state.nextId, originalAttributes: originalAttributes}
	state.nextId++
	state.template.addInstruction(&repeat)
	state.appendAction(getTalRepeatEndAction(state.template, &repeat, len(state.template.instructions)-1))
	return nil
}

/*
getTalRepeatEndAction is used for the end tag of a tal:repeat.

The offset from the renderRepeat instruction is calculated and a new
renderEndRepeat template instruction is created to perform the loop.
*/
func getTalRepeatEndAction(t *Template, repeat *renderRepeat, startRepeatIndex int) endActionFunc {
	return func() {
		// Let the start tag know where the end tag is.
		repeat.endTagOffset = len(t.instructions) - startRepeatIndex - 1
		// Add a end of repeat instruction.
		endRepeat := &renderEndRepeat{repeatName: repeat.repeatName, repeatId: repeat.repeatId, repeatStartOffset: -1 * (repeat.endTagOffset + 1)}
		t.addInstruction(endRepeat)
	}
}

/*
talOmitTagStart is used for tal:omit.

talStartTag is changed as required to implement omit-tag.
No endAction is used.
*/
func talOmitTagStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	if len(talValue) == 0 {
		return state.error(ErrExpressionMissing)
	}
	state.talEndTag.checkOmitTagFlag = true
	state.talStartTag.omitTagExpression = talValue
	return nil
}

/*
getPlainEndTagAction is used for the end tags without tal or metal commands.

Template.addRenderInstruction is used in case the render template instruction
can be coalesced with a previous render command.
*/
func getPlainEndTagAction(t *Template, tagName []byte) endActionFunc {
	return func() {
		var d buffer
		d.appendString("</")
		d.append(tagName)
		d.appendString(">")
		// Use a plain renderData for end tags that have no TAL associated with them.
		t.addRenderInstruction(d)
	}
}

/*
getTalEndTagAction returns an endAction that completes setting up the start tag
and adds the end tag instruction.

This function is inserted into the start of the list of end actions, so all
end actions done after this need to consider that the renderEndTag instruction
has already been added to the template.
*/
func getTalEndTagAction(currentStartTag *renderStartTag, currentEndTag *renderEndTag, t *Template) endActionFunc {
	startLocation := len(t.instructions) - 1
	return func() {
		// This action is executed once the end tag is seen
		currentStartTag.endTagOffset = len(t.instructions) - startLocation

		// Now add the end tag if appropriate
		if !currentStartTag.voidElement {
			t.instructions = append(t.instructions, currentEndTag)
		}
	}
}

/*
CompileTemplate reads the template in and compiles it ready for execution.

If a compilation error (rather than IO error) occurs, the returned error
will be a CompileError object.
*/
func CompileTemplate(in io.Reader) (template *Template, err error) {
	tokenizer := html.NewTokenizer(in)
	template = newTemplate()
	state := &compileState{template: template, tokenizer: tokenizer}

	for {
		token := tokenizer.Next()
		switch token {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return template, nil
			}
			return nil, tokenizer.Err()
		case html.TextToken:
			var d buffer
			// Text() returns a []byte that may change, so we immediately make a copy
			d.appendString(html.EscapeString(string(tokenizer.Text())))
			template.addRenderInstruction(d)
		case html.StartTagToken:
			rawTagName, hasAttr := tokenizer.TagName()
			// rawTagName is a slice of bytes that may change when next() is called on the tokenizer.
			// To avoid subtle bugs we create a copy of the data that we know will be immutable
			tagName := make([]byte, len(rawTagName))
			copy(tagName, rawTagName)
			// Note the tag
			var voidElement bool = htmlVoidElements[string(tagName)]
			state.addTag(tagName)

			var d buffer
			var originalAtts []html.Attribute
			var talAtts []html.Attribute
			var key, val, rawkey, rawval []byte
			for hasAttr {
				rawkey, rawval, hasAttr = tokenizer.TagAttr()
				// TagAttr returns slides that may change when next() is called - so duplicate them before casting to string
				key = make([]byte, len(rawkey))
				copy(key, rawkey)
				val = make([]byte, len(rawval))
				copy(val, rawval)
				att := html.Attribute{Key: string(key), Val: string(val)}
				if strings.HasPrefix(att.Key, "tal:") || strings.HasPrefix(att.Key, "metal:") {
					talAtts = append(talAtts, att)
				} else {
					originalAtts = append(originalAtts, att)
				}
			}
			if len(talAtts) == 0 {
				d.appendString("<")
				d.append(tagName)
				for _, att := range originalAtts {
					d.appendString(" ")
					d.appendString(att.Key)
					d.appendString(`="`)
					d.appendString(html.EscapeString(att.Val))
					d.appendString(`"`)
				}
				d.appendString(">")
				template.addRenderInstruction(d)

				// Register an action to add the close tag in when we see it.
				// This is done via an action so that we can use different logic for close tags that have tal commands
				// tagName is captured by the closure
				if !voidElement {
					state.appendAction(getPlainEndTagAction(template, tagName))
				} else {
					// If we have a void element, pop it off the stack straight away
					err = state.popTag(tagName)
					if err != nil {
						return nil, err
					}
				}

				break
			}

			// Empty out the start and end tag state
			state.talStartTag = &renderStartTag{tagName: tagName, originalAttributes: originalAtts, voidElement: voidElement}
			state.talEndTag = &renderEndTag{tagName: tagName, checkOmitTagFlag: false}

			// Sort the tal attributes into priority order
			sort.Sort(talAttributes(talAtts))
			// Process each one.
			for _, talCommand := range talAtts {
				properties, ok := talCommandProperties[talCommand.Key]
				if !ok {
					// As we are returning here we know that tokenizer will not get a chance to change the results of Raw() or Buffered()
					return nil, newCompileError(ErrUnknownTalCommand, state.tokenizer.Raw(), state.tokenizer.Buffered())
				}
				err := properties.StartAction(originalAtts, talCommand.Val, state)
				if err != nil {
					return nil, err
				}
			}
			// Output the start tag
			currentStartTag := state.talStartTag
			currentEndTag := state.talEndTag
			template.addInstruction(state.talStartTag)

			/*
				Register the end tag function.  This:
				Updates the start tag with the location of the end tag
				Does a special render for the end tag which know to omit itself if tal:omit is used

				currentStartTag, currentEndTag and tagName are defined inside the for loop and so are captured within the closure
			*/
			state.insertAction(getTalEndTagAction(currentStartTag, currentEndTag, template))

			/*
				If we have a void element, run through all end actions immediately.
			*/
			if currentStartTag.voidElement {
				err = state.popTag(tagName)
				if err != nil {
					return nil, err
				}
			}

		case html.EndTagToken:
			tagName, _ := tokenizer.TagName()
			// WARNING: tagName is not immutable.
			// For popTag this is not a problem, for other uses it may be.

			// Pop a tag off our stack
			err = state.popTag(tagName)
			if err != nil {
				return nil, err
			}
			//template.addRenderInstruction(d)
		case html.SelfClosingTagToken:
		case html.CommentToken:
			var d buffer
			d.appendString("<!--")
			d.appendString(html.EscapeString(string(tokenizer.Text())))
			d.appendString("-->")
			template.addRenderInstruction(d)
		case html.DoctypeToken:
			var d buffer
			d.appendString("<!DOCTYPE ")
			d.appendString(html.EscapeString(string(tokenizer.Text())))
			d.appendString(">")
			template.addRenderInstruction(d)
		}
	}
	return
}
