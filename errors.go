package halgo

import (
	"fmt"
)

type HalgoError struct {
	Message string
}

func (err HalgoError) Error() string {
	return err.Message
}

type HTTPError struct {
	StatusCode int
	HalgoError
}

func Error404(what string) HTTPError {
	return HTTPError{404, Error(what + " not found.")}
}

func Error(args ...interface{}) HalgoError {
	message := fmt.Sprint(args...)
	//Print("ERROR RAISED:", message)
	return HalgoError{message}
}

func Errorf(format string, args ...interface{}) error {
	return Error(fmt.Sprintf(format, args...))
}

func Print(args ...interface{}) {
	fmt.Println(args)
}
