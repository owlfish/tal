package tal

import ()

type stackValue struct {
	name   string
	value  interface{}
	exists bool
}

type variableContainer struct {
	values map[string]interface{}
	stack  []stackValue
}

/*
newContainer returns a ready to use variableContainer.
*/
func newContainer() *variableContainer {
	c := &variableContainer{
		values: make(map[string]interface{}),
		stack:  make([]stackValue, 0, 5),
	}
	return c
}

func (c *variableContainer) GetValue(name string) (interface{}, bool) {
	v, ok := c.values[name]
	return v, ok
}

func (c *variableContainer) AddValue(name string, value interface{}) {
	curVal, exists := c.values[name]
	c.stack = append(c.stack, stackValue{name, curVal, exists})
	c.values[name] = value
}

func (c *variableContainer) SetValue(name string, value interface{}) {
	c.values[name] = value
}

func (c *variableContainer) RemoveValue() {
	stackSize := len(c.stack)
	if stackSize == 0 {
		return
	}
	stackEntry := c.stack[stackSize-1]
	c.stack = c.stack[:stackSize-1]

	if stackEntry.exists {
		// Restore the old value
		c.values[stackEntry.name] = stackEntry.value
	} else {
		// Remove the value from the map
		delete(c.values, stackEntry.name)
	}
}
