package tal

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

/*
repeatVariable implements the tal repeat variable.

There are some differences from the spec:

1 - first and last grouping functions are not supported.

2 - LetterUpper and RomanUpper are used instead of Letter and Roman
*/
type repeatVariable struct {
	// The sequence being itterated over
	sequence interface{}
	// The reflected value of the sequence value.
	sequenceValue    reflect.Value
	sequenceLength   int
	sequencePosition int
	// repeatId is a unique ID for this template used to allow re-using repeat names
	repeatId int
}

// Index returns the current position within the sequence, starting at 0.
func (rv *repeatVariable) Index() int {
	return rv.sequencePosition
}

// Number returns the current position within the sequence, starting at 1.
func (rv *repeatVariable) Number() int {
	return rv.sequencePosition + 1
}

// Even returns true if the current iteration is an even index
func (rv *repeatVariable) Even() bool {
	return rv.sequencePosition%2 == 0
}

// Even returns ture if the current iteration is an odd index
func (rv *repeatVariable) Odd() bool {
	return rv.sequencePosition%2 != 0
}

// Start returns true if this is the first iteration
func (rv *repeatVariable) Start() bool {
	return rv.sequencePosition == 0
}

// Start returns true if this is the last iteration
func (rv *repeatVariable) End() bool {
	return rv.sequencePosition == rv.sequenceLength-1
}

// Length returns the total number of iterations
func (rv *repeatVariable) Length() int {
	return rv.sequenceLength
}

// Letter returns a letter (a, b, etc) for the iteration
func (rv *repeatVariable) Letter() string {
	var result string
	value := rv.sequencePosition

	for value >= 0 {
		thisColumn := value % 26
		value = value / 26
		value-- // Required because there is no zero in the letter sequence.
		result = string('a'+thisColumn) + result
	}
	return result
}

// LetterUpper returns the upper case version of Letter
func (rv *repeatVariable) LetterUpper() string {
	return strings.ToUpper(rv.Letter())
}

// Roman returns the Number as roman numerals.
func (rv *repeatVariable) Roman() string {
	romanNumeralList := []struct {
		numeral string
		value   int
	}{
		{"m", 1000},
		{"cm", 900},
		{"d", 500},
		{"cd", 400},
		{"c", 100},
		{"xc", 90},
		{"l", 50},
		{"xl", 40},
		{"x", 10},
		{"ix", 9},
		{"v", 5},
		{"iv", 4},
		{"i", 1},
	}

	// Roman numbers only supported up to 4000
	if rv.sequencePosition > 3999 {
		return " "
	}

	number := rv.sequencePosition + 1
	var result string
	for _, roman := range romanNumeralList {
		for number >= roman.value {
			result += roman.numeral
			number -= roman.value
		}
	}
	return result
}

// RomanUpper returns the upper case version of Roman
func (rv *repeatVariable) RomanUpper() string {
	return strings.ToUpper(rv.Roman())
}

// indexedValue returns the current value
func (rv *repeatVariable) indexedValue() interface{} {
	return rv.sequenceValue.Index(rv.sequencePosition).Interface()
}

// newRepeatVariable creates a new repeat variable.
func newRepeatVariable(repeatID int, sequence interface{}) *repeatVariable {
	rv := &repeatVariable{}
	rv.sequence = sequence
	rv.sequenceValue = reflect.Indirect(reflect.ValueOf(sequence))
	rv.sequenceLength = rv.sequenceValue.Len()
	rv.repeatId = repeatID
	return rv
}

// tales holds the state used when evaluating tales expressions.
type tales struct {
	// data holds the user provided data
	data interface{}
	// localVariables holds all currently defined local variables
	localVariables *variableContainer
	// globalVariables holds all currently defined global variables
	globalVariables *variableContainer
	// repeatVariables holds any defined repeat variables
	repeatVariables *variableContainer
	// debug holds the function to use for debug logging
	debug LogFunc
	// originalAttributes holds the attributes of the current element
	originalAttributes attributesList
}

/*
Default is a special value used by TAL to indicate that the default template content should be used in tal:content, etc.

Use the top level TALES variable "default" for the default value in a path.

For None use the path "nothing" and in Go use nil.
*/
var Default interface{} = struct{ Name string }{"Default"}

// notFound is returned internally during path resolution if a property can not be found.
var notFound interface{} = struct{ Name string }{"Not found"}

/*
trueOfFalse determines whether a TALES value is ture or false.

Empty strings, integers and floats of 0 value, empty slices and false booleans are all false.

Any other value is true.
*/
func trueOrFalse(value interface{}) bool {
	if value == nil || value == notFound {
		return false
	}
	switch a := value.(type) {
	case string:
		if len(a) == 0 {
			return false
		}
	case int:
		if a == 0 {
			return false
		}
	case float32:
		if a == 0 {
			return false
		}
	case float64:
		if a == 0 {
			return false
		}
	case bool:
		return a
	}
	// Check whether the value is a sequence
	reflectValue := reflect.Indirect(reflect.ValueOf(value))
	if reflectValue.Kind() == reflect.Slice {
		if reflectValue.Len() == 0 {
			return false
		}
	}
	return true
}

// isValueSequence returns true if the value can be used as a sequence, i.e is
// a slice or an array.
func isValueSequence(value interface{}) bool {
	a := reflect.ValueOf(value)
	if a.Kind() == reflect.Slice {
		return true
	}
	return a.Kind() == reflect.Array
}

/*
evaluate takes a TALES expression and returns it's result.
*/
func (t *tales) evaluate(talesExpression string, originalAttributes attributesList) interface{} {
	// Figure out what kind of expression we have
	t.originalAttributes = originalAttributes
	result := t.evaluateExpression(talesExpression)
	t.debug("TALES evaluated %v to value %v\n", talesExpression, result)
	return result
}

/*
evaluateExpression can be recursively called and evalutes a TALES expression.

This is used to evaluate multiple | separated expressions.
*/
func (t *tales) evaluateExpression(talesExpression string) interface{} {
	// Figure out what kind of expression we have
	talesExpression = strings.TrimSpace(talesExpression)

	if strings.HasPrefix(talesExpression, "path:") {
		value := t.evaluatePath(talesExpression[5:])
		if value == notFound {
			value = nil
		}
		return value
	} else if strings.HasPrefix(talesExpression, "string:") {
		return t.evaluteStringExpression(talesExpression[7:])
	} else if strings.HasPrefix(talesExpression, "exists:") {
		// Exists applies to paths, not expressions.
		value := t.evaluatePath(talesExpression[7:])
		if value == notFound {
			return false
		}
		return true
	} else if strings.HasPrefix(talesExpression, "not:") {
		// Not applies to expressions, not paths
		value := t.evaluateExpression(talesExpression[4:])
		return !trueOrFalse(value)
	} else {
		// No prefix - treat as a path expression.
		value := t.evaluatePath(talesExpression)
		if value == notFound {
			value = nil
		}
		return value
	}
	return nil
}

/*
evaluteStringExpression implements TALES string: expressions.
*/
func (t *tales) evaluteStringExpression(expression string) string {
	expression = strings.TrimSpace(expression)
	chars := []rune(expression)
	length := len(chars)
	var output buffer = make(buffer, 0, len(chars)*2)
	var position, handled int
	var foundDollar, inBrackets bool
	for position < length {
		char := chars[position]
		switch char {
		case '$':
			if foundDollar {
				// We've found a second dollar - are they back to back?
				if handled == position {
					output.appendString("$")
					foundDollar = false
				} else {
					// Treat as the end of a variable
					value := t.evaluatePath(string(chars[handled:position]))
					output.appendString(fmt.Sprint(value))
					foundDollar = true
				}
				handled = position + 1
			} else {
				// First dollar - output any normal text so far
				if handled < position-1 {
					output.appendString(string(chars[handled:position]))
				}
				handled = position + 1
				foundDollar = true
			}
		case ' ':
			if foundDollar {
				// End of the variable name - look it up.
				value := t.evaluatePath(string(chars[handled:position]))
				output.appendString(fmt.Sprint(value))
				handled = position
				foundDollar = false
			}
		case '{':
			inBrackets = true
		case '}':
			if inBrackets {
				value := t.evaluatePath(string(chars[handled+1 : position]))
				output.appendString(fmt.Sprint(value))
				handled = position + 1
				inBrackets = false
				foundDollar = false
			}
		}
		position++
	}
	// See if we have any end of string terminated variables
	if foundDollar {
		// Last variable - expand it.
		t.debug("String tales path looking for %v at end of loop\n", string(chars[handled:]))
		value := t.evaluatePath(string(chars[handled:]))
		output.appendString(fmt.Sprint(value))
	} else {
		// Finish off any remaining output
		output.appendString(string(chars[handled:]))
	}
	return string(output)
}

/*
expandPathSegment checks for variable path segments (?segment) and expands the variable if required.

The result will contain the path segment to use after expansion.  If expansion fails then an empty string is returned.
*/
func (t *tales) expandPathSegment(segment string) (result string) {
	if len(segment) < 2 {
		// Not long enough to hold a variable
		return segment
	}
	if segment[0] != '?' {
		// Not a variable - return as-is
		return segment
	}
	// segment is a variable reference - need to expand it
	segmentValue := t.evaluatePath(segment[1:])
	if segmentValue == nil || segmentValue == Default || segmentValue == notFound {
		return ""
	}
	switch a := segmentValue.(type) {
	case string:
		return a
	case int:
		// Cast to string
		return strconv.Itoa(a)
	}
	return ""
}

/*
evaluatePath evaluates a path: or implied TALES path expression.

The | operator is supported, triggering recursive calls to evaluateExpression.
*/
func (t *tales) evaluatePath(talesExpression string) interface{} {
	// Do we have alternative expressions to evaluate?
	talesExpression = strings.TrimSpace(talesExpression)

	endOfExpression := strings.Index(talesExpression, "|")
	pathExpression := talesExpression

	if endOfExpression > -1 {
		// We have a path and then one or more alternative expressions, e.g. path:a/b | string: Hello
		// Only evaluate the first path we have
		pathExpression = talesExpression[:endOfExpression]
	}

	// We need to figure out the root object (local, global, user, repeat) before we can evaluate further
	// Breakup the path
	pathElements := strings.Split(pathExpression, "/")
	if len(pathElements) == 0 {
		// This should never happen
		return notFound
	}

	objectName := pathElements[0]

	// Special values.
	if objectName == "nothing" {
		return nil
	}
	if objectName == "default" {
		return Default
	}

	if objectName == "attrs" {
		// Looking for an original attribute value
		if len(pathElements) < 2 {
			// In case the template does something silly like: attrs |  string: No repeat, we should check and act on any remaining expressions
			if endOfExpression > -1 {
				return t.evaluateExpression(talesExpression[endOfExpression+1:])
			}
			// If this is the last expression being evaluated - return None
			return nil
		}
		expandedPathElement := t.expandPathSegment(pathElements[1])
		if expandedPathElement == "" {
			return notFound
		}
		return t.originalAttributes.Get(expandedPathElement)
	}

	if objectName == "repeat" {
		// Looking for a repeat variable
		if len(pathElements) < 2 {
			// In case the template does something silly like: repeat |  string: No repeat, we should check and act on any remaining expressions
			if endOfExpression > -1 {
				return t.evaluateExpression(talesExpression[endOfExpression+1:])
			}
			// If this is the last expression being evaluated - return None
			return nil
		}
		expandedPathElement := t.expandPathSegment(pathElements[1])
		if expandedPathElement == "" {
			return notFound
		}
		value, ok := t.repeatVariables.GetValue(expandedPathElement)
		if ok {
			t.debug("Found repeat variable %v - resolve remaining path parts %v\n", pathElements[1], pathElements[2:])
		} else {
			t.debug("Unable to find repeat variable %v - returning not found\n", pathElements[1])
			return notFound
		}
		return t.resolvePathObject(value, pathElements[2:])
	}

	// Check local variables next
	value, ok := t.localVariables.GetValue(objectName)
	if ok {
		pathValue := t.resolvePathObject(value, pathElements[1:])
		if pathValue == notFound && endOfExpression > -1 {
			return t.evaluateExpression(talesExpression[endOfExpression+1:])
		}
		return pathValue
	}

	// Check the global variables
	value, ok = t.globalVariables.GetValue(objectName)
	if ok {
		pathValue := t.resolvePathObject(value, pathElements[1:])
		if pathValue == notFound && endOfExpression > -1 {
			return t.evaluateExpression(talesExpression[endOfExpression+1:])
		}
		return pathValue
	}

	// Try the user provided data
	pathValue := t.resolvePathObject(t.data, pathElements)
	if pathValue == notFound && endOfExpression > -1 {
		return t.evaluateExpression(talesExpression[endOfExpression+1:])
	}
	return pathValue
}

/*
resolvePathObject takes an object and traverses the path to get it's property.

The object will have been found in local, global or user data.  Properties
will be traversed, including maps, structs, functions and methods to get to
a final object.
*/
func (t *tales) resolvePathObject(value interface{}, path []string) interface{} {
	candidate := value
	for _, property := range path {
		propertyExpanded := t.expandPathSegment(property)
		if propertyExpanded == "" {
			return notFound
		}
		candidate = t.resolveObjectProperty(candidate, propertyExpanded)
		if candidate == notFound {
			// If the property can't be found - return it
			return notFound
		}
		if candidate == nil {
			// If the candidate resolve to None there are no attributes, just return it
			return nil
		}
	}
	return candidate
}

// callMethod attempts to call the given named property as a method.
// A single return value is supported.
func (t *tales) callMethod(data reflect.Value, goFieldName string) (result interface{}) {
	// If calling the method panics, recover
	defer func() {
		if recover() != nil {
			result = notFound
		}
	}()

	method := data.MethodByName(goFieldName)
	t.debug("Result of looking for method %v: %v\n", goFieldName, method)
	if method.IsValid() {
		t.debug("Found method in struct, calling.\n")
		var callArgs []reflect.Value = make([]reflect.Value, 0, 0)
		results := method.Call(callArgs)
		if len(results) > 0 {
			return results[0].Interface()
		}
		return nil
	}
	return notFound
}

// callFunc attempts to call the function provided.
// A single return value is supported.
func (t *tales) callFunc(data reflect.Value) (result interface{}) {
	// If calling the function panics, recover
	defer func() {
		if recover() != nil {
			result = notFound
		}
	}()

	var callArgs []reflect.Value = make([]reflect.Value, 0, 0)
	results := data.Call(callArgs)
	if len(results) > 0 {
		return results[0].Interface()
	}
	return nil
}

/*
resolveObjectProperty takes a single value and returns a named property.

For maps the property is treated as a key.  For structs the property has
it's first letter made upper case (i.e. exported) and are looked for in
fields and methods.

Any func or method found will be called and it's value will be returned.
*/
func (t *tales) resolveObjectProperty(value interface{}, property string) interface{} {
	rawData := reflect.ValueOf(value)
	data := reflect.Indirect(rawData)
	kind := data.Kind()
	propertyValue := reflect.ValueOf(property)
	t.debug("Looking for property %v in data %v (kind %v)\n", property, value, kind)
	switch kind {
	case reflect.Map:
		// Lookup the value
		mapResult := data.MapIndex(propertyValue)
		if mapResult.IsValid() {
			t.debug("TALES: Found value in map\n")
			mapValue := mapResult.Interface()
			// Look at the value
			mapValueReflection := reflect.ValueOf(mapValue)

			if mapValueReflection.Kind() == reflect.Func {
				t.debug("Found function - calling it.\n")
				return t.callFunc(mapValueReflection)
			}
			return mapValue
		}
		return notFound
	case reflect.Struct:
		// Lookup the value
		// Go field names start with upper case to be exported
		goFieldName := strings.ToUpper(property[:1]) + property[1:]
		structField := data.FieldByName(goFieldName)
		if structField.IsValid() {
			// Make it concerete if it's an interface
			// if structField.Kind() == reflect.Interface {
			// 	structField = reflect.ValueOf(structField)
			// }
			t.debug("TALES: Found field in struct - kind of %v\n", structField.Kind())
			// Check that this is an exported field
			if structType, _ := data.Type().FieldByName(goFieldName); structType.PkgPath == "" {
				//t.debug("TALES: Confirmed field in struct is exported - it's kind is %v\n", structField.Kind())
				// Get field value wrapped in an interface{}
				structFieldInterface := structField.Interface()
				// Now get the reflected value of this interface
				structField = reflect.ValueOf(structFieldInterface)
				t.debug("New field kind: %v\n", structField.Kind())
				if structField.Kind() == reflect.Func {
					t.debug("Found function - calling it.\n")
					return t.callFunc(structField)
				}
				return structFieldInterface
			}
		} else {
			// Start by looking for pointer methods.
			if rawData != data {
				result := t.callMethod(rawData, goFieldName)
				if result != notFound {
					return result
				}
			}
			// Now call value methods
			result := t.callMethod(data, goFieldName)
			if result != notFound {
				return result
			}
		}
		// Not a struct field or method - return not found
		return notFound
	}
	// Not a map or struct - return notFound
	return notFound

}

// newTalesContext sets up a new tales object with the given user data.
func newTalesContext(data interface{}) *tales {
	t := &tales{
		data:            data,
		localVariables:  newContainer(),
		globalVariables: newContainer(),
		repeatVariables: newContainer(),
		debug:           defaultLogger,
	}

	return t
}
