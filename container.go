package tal

import ()

type stackValue struct {
	name   string
	value  interface{}
	exists bool
}

type variableContainer struct {
	values    map[string]interface{}
	stack     []stackValue
	saveStack []map[string]interface{}
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

/*
SaveAll takes a snapshot of the current state of all variables and stores it for later restoring.

This is used for macros and fill slots that wish to preseve any global variables.
*/
func (c *variableContainer) SaveAll() {
	newMap := make(map[string]interface{})
	for k, v := range c.values {
		newMap[k] = v
	}
	c.saveStack = append(c.saveStack, c.values)
	c.values = newMap
}

/*
RestoreAll rolls back to a previous snapshot created with SaveAll.
*/
func (c *variableContainer) RestoreAll() {
	stackSize := len(c.saveStack)
	if stackSize == 0 {
		return
	}
	stackEntry := c.saveStack[stackSize-1]
	c.saveStack = c.saveStack[:stackSize-1]
	c.values = stackEntry
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
