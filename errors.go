package tal

import (
	"fmt"
)

type CompileError struct {
	LastToken string
	NextData  string
	ErrorType int
}

func (err *CompileError) Error() string {
	var msg string
	switch err.ErrorType {
	case ErrUnexpectedCloseTag:
		msg = "Unexpected Close Tag"
	default:
		msg = "Unexpected error"
	}
	return fmt.Sprintf(`Tal compilation error (%v) at "%v" prior to "%v"\n`, msg, err.LastToken, err.NextData)
}

const (
	ErrUnexpectedCloseTag = iota
	ErrUnknownTalCommand
	ErrExpressionMalformed
	ErrExpressionMissing
)

func newCompileError(errType int, lastToken []byte, nextData []byte) *CompileError {
	err := &CompileError{}
	err.LastToken = string(lastToken)
	err.NextData = string(nextData[:100])
	err.ErrorType = errType
	return err
}
