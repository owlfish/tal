package tal

import (
	"reflect"
	"strings"
)

type repeatVariable struct {
	sequence         interface{}
	sequenceValue    reflect.Value
	sequenceLength   int
	sequencePosition int
	repeatId         int
}

func (rv *repeatVariable) Index() int {
	return rv.sequencePosition
}

func (rv *repeatVariable) indexedValue() interface{} {
	return rv.sequenceValue.Index(rv.sequencePosition).Interface()
}

func newRepeatVariable(repeatID int, sequence interface{}) *repeatVariable {
	rv := &repeatVariable{}
	rv.sequence = sequence
	rv.sequenceValue = reflect.Indirect(reflect.ValueOf(sequence))
	rv.sequenceLength = rv.sequenceValue.Len()
	rv.repeatId = repeatID
	return rv
}

type tales struct {
	data            interface{}
	dataValue       reflect.Value
	localVariables  *variableContainer
	globalVariables *variableContainer
	repeatVariables *variableContainer
	debug           LogFunc
}

var None interface{} = &struct{ Name string }{Name: "None"}

var Default interface{} = &struct{ Name string }{Name: "Default"}

func trueOrFalse(value interface{}) bool {
	if value == None || value == nil {
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
	case bool:
		return a
	}
	return true
}

func isValueSequence(value interface{}) bool {
	a := reflect.ValueOf(value)
	return a.Kind() == reflect.Slice
}

func (t *tales) evaluate(talesExpression string) interface{} {
	// Figure out what kind of expression we have
	talesExpression = strings.TrimSpace(talesExpression)

	if strings.HasPrefix(talesExpression, "path:") {
		return t.evaluatePath(talesExpression[5:])
	} else if strings.HasPrefix(talesExpression, "string:") {
	} else {
		// No prefix - treat as a path expression.
		return t.evaluatePath(talesExpression)
	}
	return None
}

func (t *tales) evaluatePath(talesExpression string) interface{} {
	// Do we have alternative expressions to evaluate?
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
		return None
	}

	objectName := pathElements[0]
	if objectName == "repeat" {
		// Looking for a repeat variable
		if len(pathElements) < 2 {
			// In case the template does something silly like: repeat |  string: No repeat, we should check and act on any remaining expressions
			if endOfExpression > -1 {
				return t.evaluate(talesExpression[endOfExpression+1:])
			}
			// If this is the last expression being evaluated - return None
			return None
		}
		value, ok := t.repeatVariables.GetValue(pathElements[1])
		if ok {
			t.debug("Found repeat variable %v - resolve remaining path parts %v\n", pathElements[1], pathElements[2:])
		} else {
			t.debug("Unable to find repeat variable %v - returning None\n", pathElements[1])
			return None
		}
		return t.resolvePathObject(value, pathElements[2:])
	}

	// Check local variables next
	value, ok := t.localVariables.GetValue(objectName)
	if ok {
		pathValue := t.resolvePathObject(value, pathElements[1:])
		if pathValue == None && endOfExpression > -1 {
			return t.evaluate(talesExpression[endOfExpression+1:])
		}
		return pathValue
	}

	// Check the global variables
	value, ok = t.globalVariables.GetValue(objectName)
	if ok {
		pathValue := t.resolvePathObject(value, pathElements[1:])
		if pathValue == None && endOfExpression > -1 {
			return t.evaluate(talesExpression[endOfExpression+1:])
		}
		return pathValue
	}

	// Try the user provided data
	pathValue := t.resolvePathObject(t.data, pathElements)
	if pathValue == None && endOfExpression > -1 {
		return t.evaluate(talesExpression[endOfExpression+1:])
	}
	return pathValue
}

func (t *tales) resolvePathObject(value interface{}, path []string) interface{} {
	candidate := value
	for _, property := range path {
		candidate = t.resolveObjectProperty(candidate, property)
		if candidate == None {
			return None
		}
	}
	return candidate
}

func (t *tales) callMethod(data reflect.Value, goFieldName string) (result interface{}) {
	method := data.MethodByName(goFieldName)
	t.debug("Result of looking for method %v: %v\n", goFieldName, method)
	if method.IsValid() {
		t.debug("Found method in struct, calling.\n")
		var callArgs []reflect.Value = make([]reflect.Value, 0, 0)
		results := method.Call(callArgs)
		if len(results) > 0 {
			return results[0].Interface()
		}
		return None
	}
	return nil
}

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
			return mapResult.Interface()
		}
		return None
	case reflect.Struct:
		// Lookup the value
		// Go field names start with upper case to be exported
		goFieldName := strings.ToUpper(property[:1]) + property[1:]
		structField := data.FieldByName(goFieldName)
		if structField.IsValid() {
			t.debug("TALES: Found field in struct\n")
			// Check that this is an exported field
			if structType, _ := data.Type().FieldByName(goFieldName); structType.PkgPath == "" {
				t.debug("TALES: Confirmed field in struct is exported\n")
				return structField.Interface()
			}
		} else {
			// Start by looking for pointer methods.
			if rawData != data {
				result := t.callMethod(rawData, goFieldName)
				if result != nil {
					return result
				}
			}
			// Now call value methods
			result := t.callMethod(data, goFieldName)
			if result != nil {
				return result
			}
		}
		return None
	}
	return None

}

func newTalesContext(data interface{}) *tales {
	t := &tales{
		data:            data,
		dataValue:       reflect.ValueOf(data),
		localVariables:  newContainer(),
		globalVariables: newContainer(),
		repeatVariables: newContainer(),
		debug:           defaultLogger,
	}

	return t
}
