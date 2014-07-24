package main

import (
	"reflect"
	"strings"
)

type Resource interface {
	ChildResources() []Resource
}

type route map[string]node

type node struct {
	getter   getter
	children route
}

func NewRoutes(root Resource) (node, error) {
	t := reflect.TypeOf(root)
	if getter, err := analyseGetter(t); err != nil {
		return node{}, err
	} else if children, err := newChildren(root.ChildResources()); err != nil {
		return node{}, err
	} else {
		return node{getter, children}, nil
	}
}

func newChildren(children []Resource) (route, error) {
	r := map[string]node{}
	for _, c := range children {
		fullname := reflect.TypeOf(c).Name()
		name := strings.ToLower(strings.TrimSuffix(fullname, "Resource"))
		if node, err := NewRoutes(c); err != nil {
			return route{}, err
		} else if _, nameAlreadyExists := r[name]; nameAlreadyExists {
			return route{}, Errorf("Conflicting routes found for %v", name)
		} else {
			r[name] = node
		}
	}
	return r, nil
}

type getter func(routeValues []string) (interface{}, error)

type standardMethod func([]string, interface{}) (interface{}, error)

var _standardMethod standardMethod

func analyseGetter(t reflect.Type) (getter, error) {
	ptr := reflect.TypeOf(newPtrTo(t))
	if method, ok := t.MethodByName("HandleGET"); !ok {
		return nil, Error(t, "does not have a method named HandleGET")
	} else if method.Type.NumOut() != 2 {
		return nil, Error(t, ".HandleGET does not have 2 outputs.")
	} else if method.Type.Out(0) != ptr {
		return nil, Error(t, ".HandleGET's first output must be ", ptr)
	} else if method.Type.Out(1).Name() != "error" {
		return nil, Error(t, ".HandleGET's second output must be error")
	} else {
		numIn := method.Type.NumIn()
		// skip the first, as that's the reciever
		for i := 1; i < numIn; i++ {
			if method.Type.In(i).Name() != "string" {
				return nil, Error(t, ".HandleGET should accept only string inputs, not ", method.Type.In(i))
			}
		}
		v := reflect.New(t).Interface()
		caller := reflect.ValueOf(v).MethodByName("HandleGET")
		getter := func(routeValues []string) (interface{}, error) {
			in := make([]reflect.Value, len(routeValues))
			for i, a := range routeValues {
				in[i] = reflect.ValueOf(a)
			}
			returned := caller.Call(in)
			var err error
			if !returned[1].IsNil() {
				err = returned[1].Interface().(error)
			}
			return returned[0].Interface(), err
		}
		return getter, nil
	}
}

func newPtrTo(t reflect.Type) interface{} {
	return reflect.New(t).Interface()
}
