package common

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

const ErrorCallerDepth = 3

type ErrorBase struct {
	s         string
	Stack     string
	Source    string
	Function  string
	Message   string
	ErrorCode int
}

type Error struct {
	ErrorBase
}

func NewError(text string) *Error {
	e := new(Error)
	e.Stack = getStackTrace()
	e.s = text
	e.SetCallerDepth(ErrorCallerDepth)
	e.ErrorCode = 0
	return e
}

func (e *Error) Error() string {
	return e.s
}

func Errorf(format string, a ...interface{}) *Error {
	return NewError(fmt.Sprintf(format, a...))
}

func (e *Error) SetCallerDepth(depth int) {
	e.Source, e.Function = SetCallerDepth(depth)
}

func SetCallerDepth(depth int) (source string, function string) {
	pc, path, line, ok := runtime.Caller(depth)
	if ok {
		parts := strings.Split(path, "/")
		count := len(parts)
		source = fmt.Sprintf("%s:%d", parts[count-2]+"/"+parts[count-1], line)
		funcPC := runtime.FuncForPC(pc)
		if funcPC != nil {
			function = filepath.Base(funcPC.Name())
		}
	}
	return source, function
}

func getStackTrace() string {
	buf := make([]byte, 1<<16)
	sz := runtime.Stack(buf, false)
	if sz != 0 {
		return string(buf[0:sz])
	}
	return ""
}

type HttpError struct {
	ErrorBase
	StatusCode int
}

func NewHttpError(text string, code int) *HttpError {
	e := new(HttpError)
	e.Stack = getStackTrace()
	e.s = text
	e.SetCallerDepth(ErrorCallerDepth)
	e.StatusCode = code
	e.ErrorCode = 0
	return e
}

func (e *HttpError) Error() string {
	return e.s
}

func HttpErrorf(code int, format string, a ...interface{}) *HttpError {
	return NewHttpError(fmt.Sprintf(format, a...), code)
}

func HttpErrorfCode(statusCode int, errorCode int, format string, a ...interface{}) *HttpError {
	e := NewHttpError(fmt.Sprintf(format, a...), statusCode)
	e.ErrorCode = errorCode
	return e
}

func (e *HttpError) SetCallerDepth(depth int) {
	e.Source, e.Function = SetCallerDepth(depth)
}
