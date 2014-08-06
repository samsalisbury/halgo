package halgo

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"reflect"
)

// TODO: Return 400 Bad Request if deserialisation fails completely
//       Invoke validation and return 422 (Maybe) if validation fails.

func prepare_payload(body io.ReadCloser, t reflect.Type) (interface{}, error) {
	v := reflect.New(t).Interface()
	defer body.Close()
	if buf, err := ioutil.ReadAll(body); err != nil {
		return nil, err
	} else if err := json.Unmarshal(buf, &v); err != nil {
		return nil, Error("Unable to deserialise: ", string(buf), " into type ", t.Name(), " ... ", err.Error())
	} else {
		return v, nil
	}
}

func json_error(err error) []byte {
	halgoErr, ok := err.(HalgoError)
	if !ok {
		halgoErr = Error(err.Error())
	}
	if buf, jsonErr := json.Marshal(halgoErr); jsonErr == nil {
		return buf
	} else {
		return []byte("Unable to serialise error: '" + err.Error() + "'")
	}
}
