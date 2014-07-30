package halgo

import (
	"testing"
)

/////

type MISSING_ERROR_OUTPUT struct{}

func (MISSING_ERROR_OUTPUT) GET() *MISSING_ERROR_OUTPUT { return nil }

func Test_MissingErrorOutput(t *testing.T) {
	_, err := buildRoutes(MISSING_ERROR_OUTPUT{})
	error_should_contain(t, err, "MISSING_ERROR_OUTPUT.GET should have 2 outputs")
}

/////

type MISSING_RESOURCE_OUTPUT struct{}

func (MISSING_RESOURCE_OUTPUT) GET() error { return nil }

func Test_MISSING_RESOURCE_OUTPUT(t *testing.T) {
	_, err := buildRoutes(MISSING_RESOURCE_OUTPUT{})
	error_should_contain(t, err, "MISSING_RESOURCE_OUTPUT.GET should have 2 outputs")
}

/////

type WRONG_FIRST_OUT_PARAMETER struct{}

func (WRONG_FIRST_OUT_PARAMETER) GET() (error, error) { return nil, nil }

func Test_WRONG_FIRST_OUT_PARAMETER(t *testing.T) {
	_, err := buildRoutes(WRONG_FIRST_OUT_PARAMETER{})
	error_should_contain(t, err, "WRONG_FIRST_OUT_PARAMETER.GET first output must be *halgo.WRONG_FIRST_OUT_PARAMETER (not error)")
}

/////

type WRONG_SECOND_OUT_PARAMETER struct{}

func (WRONG_SECOND_OUT_PARAMETER) GET() (*WRONG_SECOND_OUT_PARAMETER, string) { return nil, "" }

func Test_WRONG_SECOND_OUT_PARAMETER(t *testing.T) {
	_, err := buildRoutes(WRONG_SECOND_OUT_PARAMETER{})
	error_should_contain(t, err, "WRONG_SECOND_OUT_PARAMETER.GET second output must be error")
}

/////

type FIRST_OUT_PARAM_NOT_POINTER struct{}

func (FIRST_OUT_PARAM_NOT_POINTER) GET() (FIRST_OUT_PARAM_NOT_POINTER, string) {
	return FIRST_OUT_PARAM_NOT_POINTER{}, ""
}

func Test_FIRST_OUT_PARAM_NOT_POINTER(t *testing.T) {
	_, err := buildRoutes(FIRST_OUT_PARAM_NOT_POINTER{})
	error_should_contain(t, err, "FIRST_OUT_PARAM_NOT_POINTER.GET first output must be *halgo.FIRST_OUT_PARAM_NOT_POINTER (not halgo.FIRST_OUT_PARAM_NOT_POINTER)")
}

/////

type TOO_MANY_GET_PARAMS struct{}

func (TOO_MANY_GET_PARAMS) GET(a int, b int, c int, d int) (*TOO_MANY_GET_PARAMS, error) {
	return nil, nil
}

func Test_TOO_MANY_GET_PARAMS(t *testing.T) {
	_, err := buildRoutes(TOO_MANY_GET_PARAMS{})
	error_should_contain(t, err, "TOO_MANY_GET_PARAMS.GET may accept at most 2 parameters")
}

/////

type PARAMS_WRONG_ORDER struct{}

func (PARAMS_WRONG_ORDER) GET(string, map[string]string) (*PARAMS_WRONG_ORDER, error) {
	return nil, nil
}

func Test_PARAMS_WRONG_ORDER(t *testing.T) {
	_, err := buildRoutes(PARAMS_WRONG_ORDER{})
	error_should_contain(t, err, "PARAMS_WRONG_ORDER.GET Parameters out of order. Correct order is: (parentIDs map[string]string, id string)")
}

/////

type SINGLE_ID_PARAM struct{}

func (SINGLE_ID_PARAM) GET(string) (*SINGLE_ID_PARAM, error) {
	return nil, nil
}

func Test_SINGLE_ID_PARAM(t *testing.T) {
	_, err := buildRoutes(SINGLE_ID_PARAM{})
	error_should_be_nil(t, err)
}

///

type SINGLE_PARENT_IDS_PARAM struct{}

func (SINGLE_PARENT_IDS_PARAM) GET(map[string]string) (*SINGLE_PARENT_IDS_PARAM, error) {
	return nil, nil
}

func Test_SINGLE_PARENT_IDS_PARAM(t *testing.T) {
	_, err := buildRoutes(SINGLE_PARENT_IDS_PARAM{})
	error_should_be_nil(t, err)
}
