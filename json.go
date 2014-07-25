package halgo

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"reflect"
)

func prepare_payload(body io.ReadCloser, t reflect.Type) (interface{}, error) {
	v := newPtrTo(t)
	defer body.Close()
	if buf, err := ioutil.ReadAll(body); err != nil {
		return nil, err
	} else if err := json.Unmarshal(buf, v); err != nil {
		return nil, err
	} else {
		return v, nil
	}
}
