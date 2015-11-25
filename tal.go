package tal

import (
	"bytes"
	"golang.org/x/net/html"
	"io"
	"log"
	"sort"
	"strings"
)

type endActionFunc func(state *compileState)

type startActionFunc func(originalAttributes []html.Attribute, talValue string, state *compileState)

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
}

/*
addTag adds the given tag name to the stack of seen tags.
*/
func (state *compileState) addTag(tag []byte) {
	state.tagStack = append(state.tagStack, tagInfo{tag: tag})
}

/*
pushAction assocaites an action to be taken when the tag is pop'd from the stack
*/
func (state *compileState) pushAction(action endActionFunc) {
	state.tagStack[len(state.tagStack)-1].popActions = append(state.tagStack[len(state.tagStack)-1].popActions, action)
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
			act(state)
		}
		if bytes.Equal(candidate.tag, tag) {
			return nil
		}
		log.Printf("Mis-Matched tags %s and %s\n", candidate.tag, tag)
	}
	return newCompileError(ErrUnexpectedCloseTag, state.tokenizer.Raw(), state.tokenizer.Buffered())
}

type talAttributes []html.Attribute

var talCommandProperties = map[string]struct {
	Priority    int
	StartAction startActionFunc
}{
	"tal:define":     {0, unimplementedCommand},
	"tal:condition":  {1, talConditionStart},
	"tal:repeat":     {2, unimplementedCommand},
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

func unimplementedCommand(originalAttributes []html.Attribute, talValue string, state *compileState) {
	state.template.addRenderInstruction([]byte("Unimplemented tal command."))
}

func talReplaceStart(originalAttributes []html.Attribute, talValue string, state *compileState) {
	state.talStartTag.replaceCommand = true
	state.talStartTag.contentExpression = talValue
}

func talContentStart(originalAttributes []html.Attribute, talValue string, state *compileState) {
	state.talStartTag.replaceCommand = false
	state.talStartTag.contentExpression = talValue
}

func talConditionStart(originalAttributes []html.Attribute, talValue string, state *compileState) {
	condition := renderCondition{condition: talValue}
	state.template.addInstruction(&condition)
	state.pushAction(func(state *compileState) {
		// This action is executed once the end tag is seen
		// condition and state is captured inside this closure
		condition.endTagIndex = len(state.template.instructions)
	})
}

func talOmitTagStart(originalAttributes []html.Attribute, talValue string, state *compileState) {
	state.talEndTag.checkOmitTagFlag = true
	state.talStartTag.omitTagExpression = talValue
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
			var d []byte
			// Text() returns a []byte that may change, so we immediately make a copy
			d = append(d, tokenizer.Text()...)
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

			var d []byte
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
				d = append(d, []byte("<")...)
				d = append(d, tagName...)
				for _, att := range originalAtts {
					d = append(d, []byte(" ")...)
					d = append(d, []byte(att.Key)...)
					d = append(d, []byte(`="`)...)
					d = append(d, []byte(html.EscapeString(att.Val))...)
					d = append(d, []byte(`"`)...)
				}
				d = append(d, []byte(">")...)
				template.addRenderInstruction(d)

				// Register an action to add the close tag in when we see it.
				// This is done via an action so that we can use different logic for close tags that have tal commands
				// tagName is captured by the closure
				if !htmlVoidElements[string(tagName)] {
					state.pushAction(func(state *compileState) {
						var d []byte
						d = append(d, []byte("</")...)
						d = append(d, tagName...)
						d = append(d, []byte(">")...)
						// Use a plain renderData for end tags that have no TAL associated with them.
						template.addRenderInstruction(d)
					})
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
			state.pushAction(func(st *compileState) {
				// This action is executed once the end tag is seen
				// currentStartTag is captured inside this closure
				currentStartTag.endTagIndex = len(st.template.instructions)

				// Now add the end tag if appropriate
				if !currentStartTag.voidElement {
					template.instructions = append(template.instructions, currentEndTag)
				}
			})

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
			var d []byte
			d = append(d, []byte("<!--")...)
			d = append(d, tokenizer.Text()...)
			d = append(d, []byte("-->")...)
			template.addRenderInstruction(d)
		case html.DoctypeToken:
			var d []byte
			d = append(d, []byte("<!DOCTYPE ")...)
			d = append(d, tokenizer.Text()...)
			d = append(d, []byte(">")...)
			template.addRenderInstruction(d)
		}
	}
	return
}
