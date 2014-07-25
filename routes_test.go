package main

import (
	"testing"
)

/////

type MISSING_ERROR_OUTPUT struct{ test_base }

func (MISSING_ERROR_OUTPUT) HandleGET() *MISSING_ERROR_OUTPUT { return nil }

func Test_MissingErrorOutput(t *testing.T) {
	_, err := newNode(MISSING_ERROR_OUTPUT{})
	error_should_contain(t, err, "MISSING_ERROR_OUTPUT.HandleGET should have 2 outputs")
}

/////

type MISSING_RESOURCE_OUTPUT struct{ test_base }

func (MISSING_RESOURCE_OUTPUT) HandleGET() error { return nil }

func Test_MISSING_RESOURCE_OUTPUT(t *testing.T) {
	_, err := newNode(MISSING_RESOURCE_OUTPUT{})
	error_should_contain(t, err, "MISSING_RESOURCE_OUTPUT.HandleGET should have 2 outputs")
}

/////

type WRONG_FIRST_OUT_PARAMETER struct{ test_base }

func (WRONG_FIRST_OUT_PARAMETER) HandleGET() (error, error) { return nil, nil }

func Test_WRONG_FIRST_OUT_PARAMETER(t *testing.T) {
	_, err := newNode(WRONG_FIRST_OUT_PARAMETER{})
	error_should_contain(t, err, "WRONG_FIRST_OUT_PARAMETER.HandleGET first output must be *WRONG_FIRST_OUT_PARAMETER (not error)")
}

/////

type WRONG_SECOND_OUT_PARAMETER struct{ test_base }

func (WRONG_SECOND_OUT_PARAMETER) HandleGET() (*WRONG_SECOND_OUT_PARAMETER, string) { return nil, "" }

func Test_WRONG_SECOND_OUT_PARAMETER(t *testing.T) {
	_, err := newNode(WRONG_SECOND_OUT_PARAMETER{})
	error_should_contain(t, err, "WRONG_SECOND_OUT_PARAMETER.HandleGET second output must be error")
}

/////

type FIRST_OUT_PARAM_NOT_POINTER struct{ test_base }

func (FIRST_OUT_PARAM_NOT_POINTER) HandleGET() (FIRST_OUT_PARAM_NOT_POINTER, string) {
	return FIRST_OUT_PARAM_NOT_POINTER{}, ""
}

func Test_FIRST_OUT_PARAM_NOT_POINTER(t *testing.T) {
	_, err := newNode(FIRST_OUT_PARAM_NOT_POINTER{})
	error_should_contain(t, err, "FIRST_OUT_PARAM_NOT_POINTER.HandleGET first output must be *FIRST_OUT_PARAM_NOT_POINTER (not FIRST_OUT_PARAM_NOT_POINTER)")
}

/////

type TOO_MANY_GET_PARAMS struct{ test_base }

func (TOO_MANY_GET_PARAMS) HandleGET(a int, b int, c int, d int) (*TOO_MANY_GET_PARAMS, error) {
	return nil, nil
}

func Test_TOO_MANY_GET_PARAMS(t *testing.T) {
	_, err := newNode(TOO_MANY_GET_PARAMS{})
	error_should_contain(t, err, "TOO_MANY_GET_PARAMS.HandleGET may accept at most 2 parameters")
}
