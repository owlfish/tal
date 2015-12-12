package tal

import (
	"fmt"
)

/*
CompileErrors are returned from CompileTemplate for compilation errors.
*/
type CompileError struct {
	// LastToken is the last parsed HTML token seen.
	LastToken string
	// NextData contains some of the unparsed data that happens after the error.
	NextData string
	// ErrorType specifies the kind of compilation error that has occured.
	ErrorType int
}

// Error returns a text description of the compilation error.
func (err *CompileError) Error() string {
	var msg string
	switch err.ErrorType {
	case ErrUnexpectedCloseTag:
		msg = "Unexpected Close Tag"
	case ErrSlotOutsideMacro:
		msg = "metal:fill-slot used outside of macro definition"
	case ErrUnknownTalCommand:
		msg = "Unknown / unsupported command"
	case ErrExpressionMalformed:
		msg = "Parameters to tal command did not match specification."
	case ErrExpressionMissing:
		msg = "Expression missing from command"
	default:
		msg = "Unexpected error"
	}
	return fmt.Sprintf(`Tal compilation error (%v) at "%v" prior to "%v"\n`, msg, err.LastToken, err.NextData)
}

const (
	// ErrUnexpectedCloseTag is if a close tag is encountered for which an open tag was not seen.
	ErrUnexpectedCloseTag = iota
	// ErrUnknownTalCommand is if a tal: or metal: command is not one of the supported commands.
	ErrUnknownTalCommand
	// ErrExpressionMalformed is for expressions that don't match the tal command they are on.
	ErrExpressionMalformed
	// ErrExpressionMissing is if an expression is missing where one is expected.
	ErrExpressionMissing
	// ErrSlotOutsideMacro is if a metal:fill-slot is outside of a use-macro.
	ErrSlotOutsideMacro
)

// Builds a new CompileError from the data provided.
func newCompileError(errType int, lastToken []byte, nextData []byte) *CompileError {
	err := &CompileError{}
	err.LastToken = string(lastToken)
	err.NextData = string(nextData[:100])
	err.ErrorType = errType
	return err
}
