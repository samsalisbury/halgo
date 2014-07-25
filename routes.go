package main

import (
	"reflect"
	"strings"
)

const (
	handleHEAD   = "HandleHEAD"
	handleGET    = "HandleGET"
	handleDELETE = "HandleDELETE"
	handlePUT    = "HandlePUT"
	handlePATCH  = "HandlePATCH"
	handlePOST   = "HandlePOST"
)

type Resource interface {
	ChildResources() []Resource
}

type requirement int

const (
	required  requirement = iota
	optional              = iota
	forbidden             = iota
)

func (r requirement) Allowed() bool {
	return r == required || r == optional
}

type parameter_spec struct {
	ID        requirement
	ParentIDs requirement
	Payload   requirement
}

func (p parameter_spec) maxParams() int {
	n := 0
	for _, r := range []requirement{p.ID, p.ParentIDs, p.Payload} {
		if r.Allowed() {
			n++
		}
	}
	return n
}

type method func(parentID string, parentIDs map[string]string, payload interface{}) (interface{}, error)

var parameter_specs map[string]parameter_spec = map[string]parameter_spec{
	handleHEAD:   parameter_spec{optional, optional, forbidden},
	handleGET:    parameter_spec{optional, optional, forbidden},
	handleDELETE: parameter_spec{required, optional, forbidden},
	handlePUT:    parameter_spec{required, optional, required},
	handlePATCH:  parameter_spec{required, optional, required},
	handlePOST:   parameter_spec{optional, optional, required},
}

type routes map[string]node

type node struct {
	methods     map[string]method
	children    routes
	name        string
	is_identity bool
}

type methods struct {
}

func newNode(root Resource) (n node, err error) {
	t := reflect.TypeOf(root)
	if methods, err := getMethods(t); err != nil {
		return n, err
	} else if children, err := newRoutes(root.ChildResources()); err != nil {
		return n, err
	} else {
		return node{methods, children, "", false}, nil
	}
}

func newRoutes(children []Resource) (r routes, err error) {
	r = routes{}
	for _, c := range children {
		fullname := reflect.TypeOf(c).Name()
		name := strings.ToLower(strings.TrimSuffix(fullname, "Resource"))
		if node, err := newNode(c); err != nil {
			return r, err
		} else if _, nameAlreadyExists := r[name]; nameAlreadyExists {
			return r, Errorf("Conflicting routes found for %v", name)
		} else {
			node.name = name
			r[name] = node
		}
	}
	return r, nil
}

func getMethods(t reflect.Type) (map[string]method, error) {
	m := map[string]method{}
	if getter, err := analyseGetter(t); err != nil {
		return nil, err
	} else if getter != nil {
		m["GET"] = getter
	}
	// TODO: Add other methods
	return m, nil
}

func new_getter(caller reflect.Value) method {
	getter := func(id string, parentIDs []string) (interface{}, error) {
		in := make([]reflect.Value, len(parentIDs))
		for i, a := range parentIDs {
			in[i] = reflect.ValueOf(a)
		}
		returned := caller.Call(in)
		var err error
		if !returned[1].IsNil() {
			err = returned[1].Interface().(error)
		}
		return returned[0].Interface(), err
	}
	return func(id string, parentIDs map[string]string, _ interface{}) (interface{}, error) {
		return getter(id, parentIDs)
	}
}

func analyseMethodContext(t reflect.Type, name string) (method_context, bool) {
	if method_type, method_exists := t.MethodByName(name); method_exists {
		instance := reflect.New(t).Interface()
		return method_context{
			owner_pointer: reflect.TypeOf(instance),
			bound_method:  reflect.ValueOf(instance).MethodByName(name),
			method_type:   method_type,
		}
	} else {
		return method_context{}, false
	}
}

type method_context struct {
	owner_pointer_type reflect.Type
	bound_method       reflect.Value
	method_type        reflect.Type
}

type error_method func(args ...interface{}) error

func analyseGetter(t reflect.Type) (method, error) {
	E := func(args ...interface{}) error { return nil, methodError(t, handleGET, args...) }
	if ctx, ok := analyseMethodContext(t, handleGET); !ok {
		return nil, nil
	} else if err := analyseOutputs(E, ctx.method_type); err != nil {
		return nil, err
	} else if method_spec, err := analyseInputs(E, ctx.method_type, parameter_specs[handleGET]); err != nil {
		return nil, err
	} else {
		return createMethod(method_spec, ctx), nil
	}
}

type method_spec struct {
	uses_id        bool
	uses_route_ids bool
	uses_payload   bool
}

func (m method_spec) numParams() {
	n := 0
	for _, v := range []bool{m.uses_id, m.uses_route_ids, m.uses_payload} {
		if v {
			n++
		}
	}
	return n
}

func analyseInputs(E error_method, ctx method_context, p parameter_spec) (method_spec, error) {
	m := method_spec{}
	if ctx.method_type.NumIn() > p.maxParams() {
		return m, E("may accept at most", p.maxParams(), "parameters")
	}
	const musthave = "must specify a "
	const maynothave = "may not have a "
	const id_param = "string (ID) parameter"
	const parent_ids_param = "map[string]string (ParentIDs) parameter"
	var payload_param = ctx.owner_pointer_type.Name() + " (payload) parameter"
	numParams := 0
	if methodUsesID(ctx.method_type) {
		if !p.ID.Allowed() {
			return m, E(maynothave, id_param)
		} else {
			m.uses_id = true
			numParams++
		}
	} else if p.ID.Required() {
		return m, E(musthave, id_param)
	}
	if methodUsesParentIDs(ctx.method_type) {
		if !p.ParentIDs.Allowed() {
			return m, E(maynothave, parent_ids_param)
		} else {
			m.uses_route_ids = true
			numParams++
		}
	} else if p.ParentIDs.Required() {
		return m, E(musthave, parent_ids_param)
	}
	if methodUsesPayload(ctx.method_type) {
		if !p.ParentIDs.Allowed() {
			return m, E(maynothave, payload_param)
		} else {
			m.uses_payload = true
			numParams++
		}
	} else if p.Payload.Required() {
		return m, E(musthave, payload_param)
	} else {
		// for parameter := 0; len(types) > 0 && parameter < numIn {
		// 	// if method_type.In(parameter) != types[0] {
		// 	// 	types := types[1:]
		// 	// }
		// }
	}
}

func methodUsesID(method_type reflect.Type) bool {
	return methodHasParameterOfType(method_type, reflect.String)
}

func methodUsesParentIDs(method_type reflect.Type) bool {
	return methodHasParameterOfType(method_type, reflect.MapOf(reflect.String, reflect.String))
}

func methodUsesPayload(method_type reflect.Type) bool {
	return methodHasParameterOfType(method_type, reflect.Interface)
}

func methodHasParameterOfType(method_type reflect.Type, parameter_type reflect.Type) bool {
	for i = 0; i < method_type.NumIn(); i++ {
		if method_type.In(i) == reflect.String {
			return true
		}
	}
	return false
}

func analyseGetInputs(m reflect.Method) error {
	numIn = m.Type.NumIn()
	if numIn > 3 {
		return E("may accept at most 2 parameters")
	} else if numIn == 2 && m.Type.In(1) != reflect.String {
		return E("if specified, first parameter must be string (ID) or map[string]string (Parent IDs)")
	} else if numIn == 3 && m.Type.In(2) != reflect.MapOf(reflect.String, reflect.String) {
		return E("if specified, second paremeter must be map[string]string (Parent IDs)")
	}
	switch numIn {
	case 0:
		return getter_no_inputs()
	}
}

func createGetter(ctx method_context, E error_method) (method, error) {
	if err := analyseGetInputs(ctx.method_type); err != nil {
		return E(err)
	}
	caller := reflect.ValueOf(v).MethodByName(handleGET)
	getter := new_getter(caller)
	return getter, nil
}

func analyseOutputs(E error_method, m reflect.Method) error {
	if m.Type.NumOut() != 2 {
		return E("does not have 2 outputs")
	} else if m.Type.Out(0) != ptr {
		return E("first output must be ", ptr)
	} else if m.Type.Out(1).Name() != "error" {
		return E("second output must be error")
	}
	return nil
}

func routeError(args ...interface{}) {
	return Error("ROUTING: ", args)
}

func methodError(t reflect.Type, methodName string, args ...interface{}) {
	return routeError(t.Name+"."+methodName+" ", args...)
}

func newPtrTo(t reflect.Type) interface{} {
	return reflect.New(t).Interface()
}
