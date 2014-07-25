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

func Error(args ...interface{}) HalgoError {
	message := fmt.Sprint(args...)
	//Print("ERROR RAISED:", message)
	return HalgoError{message}
}

type HTTPError struct {
	StatusCode int
	HalgoError
}

func HttpError(statusCode int, args ...interface{}) HTTPError {
	return HTTPError{statusCode, Error(args...)}
}

func Error404(what string) HTTPError {
	return HttpError(404, what+" not found.")
}

func Error405(method string, n resolved_node) HTTPError {
	return HttpError(405, method+"not supported.")
}

func Errorf(format string, args ...interface{}) error {
	return Error(fmt.Sprintf(format, args...))
}

func Print(args ...interface{}) {
	fmt.Println(args)
}
