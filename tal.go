package tal

import (
	"bytes"
	"golang.org/x/net/html"
	"io"
	"log"
	"sort"
	"strings"
)

type LogFunc func(fmt string, args ...interface{})

func defaultLogger(fmt string, args ...interface{}) {

}

type endActionFunc func()

type startActionFunc func(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError

type tagInfo struct {
	tag        []byte
	popActions []endActionFunc
}

type compileState struct {
	tagStack    []tagInfo
	template    *Template
	tokenizer   *html.Tokenizer
	talStartTag *renderStartTag
	talEndTag   *renderEndTag
	nextId      int
}

/*
addTag adds the given tag name to the stack of seen tags.
*/
func (state *compileState) addTag(tag []byte) {
	state.tagStack = append(state.tagStack, tagInfo{tag: tag})
}

/*
pushAction associates an action to be taken when the tag is pop'd from the stack
*/
func (state *compileState) pushAction(action endActionFunc) {
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
popTag removes the tag from the stack and executes any deferred actions.

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
		log.Printf("Mis-Matched tags %s and %s\n", candidate.tag, tag)
	}
	return newCompileError(ErrUnexpectedCloseTag, state.tokenizer.Raw(), state.tokenizer.Buffered())
}

/*
error returns a CompileError with the context of where it happened.
*/
func (state *compileState) error(errorType int) *CompileError {
	return newCompileError(errorType, state.tokenizer.Raw(), state.tokenizer.Buffered())
}

type talAttributes []html.Attribute

/*
talCommandProperties defines the priority order of tal commands and maps them to startActionFuncs
*/
var talCommandProperties = map[string]struct {
	Priority    int
	StartAction startActionFunc
}{
	"tal:define":     {0, talDefineStart},
	"tal:condition":  {1, talConditionStart},
	"tal:repeat":     {2, talRepeatStart},
	"tal:content":    {3, talContentStart},
	"tal:replace":    {4, talReplaceStart},
	"tal:attributes": {5, unimplementedCommand},
	"tal:omit-tag":   {6, talOmitTagStart},
}

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

func splitDefineArguments(value string) []string {
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

func unimplementedCommand(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	state.template.addRenderInstruction([]byte("Unimplemented tal command."))
	return nil
}

func talDefineStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	definitionList := splitDefineArguments(talValue)
	for _, definition := range definitionList {
		if strings.HasPrefix(definition, "local ") && len(definition) > 6 {
			actualDef := strings.Split(definition[6:], " ")
			if len(actualDef) == 2 {
				state.template.addInstruction(&defineVariable{name: actualDef[0], global: false, expression: actualDef[1]})
				// Local variables need popping when the end tag is seen.
				state.pushAction(getTalDefineEndAction(state.template))
			} else {
				return state.error(ErrExpressionMissing)
			}
		} else if strings.HasPrefix(definition, "global ") && len(definition) > 7 {
			actualDef := strings.Split(definition[7:], " ")
			if len(actualDef) == 2 {
				state.template.addInstruction(&defineVariable{name: actualDef[0], global: true, expression: actualDef[1]})
			} else {
				return state.error(ErrExpressionMissing)
			}
		} else {
			// Treat as a local variable defintion.
			actualDef := strings.Split(definition, " ")
			if len(actualDef) == 2 {
				state.template.addInstruction(&defineVariable{name: actualDef[0], global: false, expression: actualDef[1]})
				// Local variables need popping when the end tag is seen.
				state.pushAction(getTalDefineEndAction(state.template))
			} else {
				return state.error(ErrExpressionMissing)
			}
		}
	}
	return nil
}

func getTalDefineEndAction(t *Template) endActionFunc {
	return func() {
		// Add a local variable remove instruction.
		t.addInstruction(&removeLocalVariable{})
	}
}

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

func talConditionStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	if len(talValue) == 0 {
		return state.error(ErrExpressionMissing)
	}
	condition := renderCondition{condition: talValue}
	state.template.addInstruction(&condition)
	state.pushAction(getTalConditionEndAction(state.template, &condition))
	return nil
}

func getTalConditionEndAction(t *Template, condition *renderCondition) endActionFunc {
	return func() {
		// This action is executed once the end tag is seen
		// condition and state is captured inside this closure
		condition.endTagIndex = len(t.instructions) - 1
	}
}

func talRepeatStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	parts := strings.Split(talValue, " ")
	if len(parts) != 2 {
		return state.error(ErrExpressionMalformed)
	}
	repeat := renderRepeat{repeatName: parts[0], condition: parts[1], repeatId: state.nextId}
	state.nextId++
	state.template.addInstruction(&repeat)
	state.pushAction(getTalRepeatEndAction(state.template, &repeat, len(state.template.instructions)-1))
	return nil
}

func getTalRepeatEndAction(t *Template, repeat *renderRepeat, startRepeatIndex int) endActionFunc {
	return func() {
		// Let the start tag know where the end tag is.
		repeat.endTagIndex = len(t.instructions) - 1
		// Add a end of repeat instruction.
		endRepeat := &renderEndRepeat{repeatName: repeat.repeatName, repeatId: repeat.repeatId, repeatStartIndex: startRepeatIndex}
		t.addInstruction(endRepeat)
	}
}

func talOmitTagStart(originalAttributes []html.Attribute, talValue string, state *compileState) *CompileError {
	if len(talValue) == 0 {
		return state.error(ErrExpressionMissing)
	}
	state.talEndTag.checkOmitTagFlag = true
	state.talStartTag.omitTagExpression = talValue
	return nil
}

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
getTalEndTagAction returns an endAction that completes setting up the start tag and adds the end tag instruction.

This function is inserted into the start of the list of end actions, so all end actions done after this need to consider
that the renderEndTag instruction has already been added to the template.
*/
func getTalEndTagAction(currentStartTag *renderStartTag, currentEndTag *renderEndTag, t *Template) endActionFunc {
	return func() {
		// This action is executed once the end tag is seen
		currentStartTag.endTagIndex = len(t.instructions)

		// Now add the end tag if appropriate
		if !currentStartTag.voidElement {
			t.instructions = append(t.instructions, currentEndTag)
		}
	}
}

func CompileTemplate(in io.Reader) (template *Template, err error) {
	tokenizer := html.NewTokenizer(in)
	template = &Template{}
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
			var voidElement bool
			if !htmlVoidElements[string(tagName)] {
				voidElement = false
				state.addTag(tagName)
			} else {
				voidElement = true
			}

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
				if strings.HasPrefix(att.Key, "tal:") {
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
				if !htmlVoidElements[string(tagName)] {
					state.pushAction(getPlainEndTagAction(template, tagName))
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
				properties.StartAction(originalAtts, talCommand.Val, state)
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
