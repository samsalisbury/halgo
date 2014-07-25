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

func (r requirement) Required() bool {
	return r == required
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

type generic_http_method func(parentID string, parentIDs map[string]string, payload interface{}) (interface{}, error)

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
	methods     map[string]*method_info
	children    routes
	name        string
	is_identity bool
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

func getMethods(t reflect.Type) (map[string]*method_info, error) {
	m := map[string]*method_info{}
	if getter, err := analyseGetter(t); err != nil {
		return nil, err
	} else if getter != nil {
		m["GET"] = getter
	}
	// TODO: Add other methods
	return m, nil
}

// func new_getter(caller reflect.Value) method {
// 	getter := func(id string, parentIDs []string) (interface{}, error) {
// 		in := make([]reflect.Value, len(parentIDs))
// 		for i, a := range parentIDs {
// 			in[i] = reflect.ValueOf(a)
// 		}
// 		returned := caller.Call(in)
// 		var err error
// 		if !returned[1].IsNil() {
// 			err = returned[1].Interface().(error)
// 		}
// 		return returned[0].Interface(), err
// 	}
// 	return func(id string, parentIDs map[string]string, _ interface{}) (interface{}, error) {
// 		return getter(id, parentIDs)
// 	}
// }

func analyseMethodContext(t reflect.Type, name string) (method_context, bool) {
	if method, method_exists := t.MethodByName(name); method_exists {
		instance := reflect.New(t).Interface()
		return method_context{
			owner_pointer_type: reflect.TypeOf(instance),
			bound_method:       reflect.ValueOf(instance).MethodByName(name),
			method_type:        method.Type,
		}, true
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

func analyseGetter(t reflect.Type) (m *method_info, err error) {
	E := func(args ...interface{}) error { return methodError(t, handleGET, args...) }
	if ctx, ok := analyseMethodContext(t, handleGET); !ok {
		return m, nil
	} else if err := analyseOutputs(E, ctx); err != nil {
		return m, err
	} else if method_spec, err := analyseInputs(E, ctx, parameter_specs[handleGET]); err != nil {
		return m, err
	} else {
		return createMethod(method_spec, ctx)
	}
}

// func analyseGetInputs(E error_method, m reflect.Method) error {
// 	numIn := m.Type.NumIn()
// 	if numIn > 3 {
// 		return E("may accept at most 2 parameters")
// 	} else if numIn == 2 && m.Type.In(1) != reflect.String {
// 		return E("if specified, first parameter must be string (ID) or map[string]string (Parent IDs)")
// 	} else if numIn == 3 && m.Type.In(2) != reflect.MapOf(reflect.String, reflect.String) {
// 		return E("if specified, second paremeter must be map[string]string (Parent IDs)")
// 	}
// 	switch numIn {
// 	case 0:
// 		return getter_no_inputs()
// 	}
// }

// func createGetter(ctx method_context, E error_method) (method, error) {
// 	if err, spec := analyseGetInputs(ctx.method_type); err != nil {
// 		return E(err)
// 	}
// 	caller := reflect.ValueOf(v).MethodByName(handleGET)
// 	getter := new_getter(caller)
// 	return getter, nil
// }

func createMethod(spec method_spec, ctx method_context) (m *method_info, err error) {
	f := func(id string, parentIDs map[string]string, payload interface{}) (interface{}, error) {
		in := make([]reflect.Value, spec.numParams())
		if spec.uses_parent_ids {
			in = append(in, reflect.ValueOf(parentIDs))
		}
		if spec.uses_id {
			in = append(in, reflect.ValueOf(id))
		}
		if spec.uses_payload {
			in = append(in, reflect.ValueOf(payload))
		}
		out := ctx.bound_method.Call(in)
		return out[0].Interface(), out[1].Interface().(error)
	}
	return &method_info{spec, ctx, f}, nil
}

type method_spec struct {
	uses_id         bool
	uses_parent_ids bool
	uses_payload    bool
}

type method_info struct {
	spec   method_spec
	ctx    method_context
	method generic_http_method
}

func (m method_spec) numParams() int {
	n := 0
	for _, v := range []bool{m.uses_id, m.uses_parent_ids, m.uses_payload} {
		if v {
			n++
		}
	}
	return n
}

// TODO: Unmentalize this function to make it more understandable
func analyseInputs(E error_method, ctx method_context, p parameter_spec) (method_spec, error) {
	m := method_spec{}
	if ctx.method_type.NumIn() > p.maxParams() {
		return m, E("may accept at most", p.maxParams(), "parameters")
	}
	const musthave = "must specify a "
	const maynothave = "may not have a "
	const id_param = "id string"
	const parent_ids_param = "parentIDs map[string]string (ParentIDs)"
	var payload_param = "payload " + ctx.owner_pointer_type.Name()
	numParams := 0
	ordering := []int{}
	// The ordering of the 3 sections below is significant.
	// It determines the accepted interfaces.
	// TODO: find a way to make the ordering explicit.
	if yes, order := methodUsesParentIDs(ctx.method_type); yes {
		if !p.ParentIDs.Allowed() {
			return m, E(maynothave, parent_ids_param, " parameter")
		}
		ordering = append(ordering, order)
		m.uses_parent_ids = true
		numParams++
	} else if p.ParentIDs.Required() {
		return m, E(musthave, parent_ids_param, " parameter")
	}
	if yes, order := methodUsesID(ctx.method_type); yes {
		if !p.ID.Allowed() {
			return m, E(maynothave, id_param, " parameter")
		}
		ordering = append(ordering, order)
		m.uses_id = true
		numParams++
	} else if p.ID.Required() {
		return m, E(musthave, id_param, " parameter")
	}
	if yes, order := methodUsesPayload(ctx.method_type, ctx.owner_pointer_type); yes {
		if !p.ParentIDs.Allowed() {
			return m, E(maynothave, payload_param, " parameter")
		}
		ordering = append(ordering, order)
		m.uses_payload = true
		numParams++
	} else if p.Payload.Required() {
		return m, E(musthave, payload_param)
	}
	if !parameter_order_correct(ordering) {
		correct_order := []string{}
		if m.uses_parent_ids {
			correct_order = append(correct_order, parent_ids_param)
		}
		if m.uses_id {
			correct_order = append(correct_order, id_param)
		}
		if m.uses_payload {
			correct_order = append(correct_order, payload_param)
		}
		return m, E("Parameters out of order. Correct order is: [" + strings.Join(correct_order, ",") + "]")
	}
	return m, nil
}

func parameter_order_correct(ordering []int) bool {
	last := 0
	for _, o := range ordering {
		if o != last+1 {
			return false
		}
	}
	return true
}

var (
	string_T = reflect.TypeOf("")
)

func methodUsesID(method_type reflect.Type) (bool, int) {
	return methodHasParameterOfType(method_type, string_T)
}

func methodUsesParentIDs(method_type reflect.Type) (bool, int) {
	return methodHasParameterOfType(method_type, reflect.MapOf(string_T, string_T))
}

func methodUsesPayload(method_type reflect.Type, payload_type reflect.Type) (bool, int) {
	return methodHasParameterOfType(method_type, payload_type)
}

func methodHasParameterOfType(method_type reflect.Type, parameter_type reflect.Type) (bool, int) {
	for i := 0; i < method_type.NumIn(); i++ {
		if method_type.In(i) == parameter_type {
			return true, i
		}
	}
	return false, 0
}

func analyseOutputs(E error_method, ctx method_context) error {
	if ctx.method_type.NumOut() != 2 {
		return E("does not have 2 outputs")
	} else if ctx.method_type.Out(0) != ctx.owner_pointer_type {
		return E("first output must be ", ctx.owner_pointer_type)
	} else if ctx.method_type.Out(1).Name() != "error" {
		return E("second output must be error")
	}
	return nil
}

func routeError(args ...interface{}) error {
	return Error("ROUTING: ", args)
}

func methodError(t reflect.Type, methodName string, args ...interface{}) error {
	prependage := t.Name() + "." + methodName + " "
	args = append([]interface{}{prependage}, args...)
	return routeError(args...)
}

func newPtrTo(t reflect.Type) interface{} {
	return reflect.New(t).Interface()
}
