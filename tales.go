package tal

import (
	"log"
	"reflect"
)

type tales struct {
	data            interface{}
	dataValue       reflect.Value
	localVariables  map[string]interface{}
	globalVariables map[string]interface{}
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

func (t *tales) evaluate(command string) interface{} {
	// Check local variables first
	value, ok := t.localVariables[command]
	if ok {
		return value
	}

	// Check the global variables
	value, ok = t.globalVariables[command]
	if ok {
		return value
	}

	// Try the user provided data
	data := t.dataValue
	data = reflect.Indirect(data)
	kind := data.Kind()
	commandValue := reflect.ValueOf(command)
	log.Printf("Looking for command %v in user data (kind %v)\n", command, kind)
	switch kind {
	case reflect.Map:
		// Lookup the value
		log.Printf("TALES: Found map\n")
		mapResult := data.MapIndex(commandValue)
		if mapResult.IsValid() {
			log.Printf("TALES: Found value in map\n")
			return mapResult.Interface()
		}
		return None
	case reflect.Struct:
		// Lookup the value
		log.Printf("TALES: Found struct\n")
		structField := data.FieldByName(command)
		// for i := 0; i < data.NumField(); i++ {
		// 	log.Printf("Stuct field %v found\n", data.Field(i))
		// }
		if structField.IsValid() {
			log.Printf("TALES: Found field in struct\n")
			// Check that this is an exported field
			if structType, _ := data.Type().FieldByName(command); structType.PkgPath == "" {
				log.Printf("TALES: Confirmed field in struct is exported\n")
				return structField.Interface()
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
		localVariables:  make(map[string]interface{}),
		globalVariables: make(map[string]interface{}),
	}

	return t
}
