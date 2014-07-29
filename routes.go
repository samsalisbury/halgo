package halgo

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

type generic_http_method func(parentIDs map[string]string, id string, payload interface{}) (interface{}, error)

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

func (n node) SupportsMethod(m string) bool {
	if _, yes := n.methods[m]; yes {
		return true
	}
	return false
}

func newNode(root Resource) (n node, err error) {
	t := reflect.TypeOf(root)
	if methods, err := getMethods(t); err != nil {
		return n, err
	} else if children, err := newRoutes(root.ChildResources()); err != nil {
		return n, err
	} else {
		n = node{methods, children, "", false}
		if yes, err := node_requires_id(n); err != nil {
			return n, err
		} else if yes {
			n.is_identity = true
		}
		return n, nil
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

type methods map[string]*method_info

func node_requires_id(n node) (bool, error) {
	really := []bool{}
	for _, n := range n.methods {
		really = append(really, n.spec.uses_id)
	}
	first_answer := really[0]
	for _, r := range really[1:] {
		if r != first_answer {
			return false, Error(n.name, "requires ID parameter in some methods but not all. This is illegal.")
		}
	}
	return first_answer, nil
}

func getMethods(t reflect.Type) (methods, error) {
	m := map[string]*method_info{}
	if getter, err := analyseGetter(t); err != nil {
		return nil, err
	} else if getter != nil {
		m["GET"] = getter
	}
	// TODO: Add other methods
	return m, nil
}

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

type error_f func(args ...interface{}) error

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

func createMethod(spec method_spec, ctx method_context) (m *method_info, err error) {
	f := func(parentIDs map[string]string, id string, payload interface{}) (interface{}, error) {
		in := make([]reflect.Value, spec.numParams())
		i := 0
		if spec.uses_parent_ids {
			println("uses_parent_ids")
			in[i] = reflect.ValueOf(parentIDs)
			i++
		}
		if spec.uses_id {
			println("uses_id")
			in[i] = reflect.ValueOf(id)
			i++
		}
		if spec.uses_payload {
			println("uses_payload")
			in[i] = reflect.ValueOf(payload)
			i++
		}
		out := ctx.bound_method.Call(in)
		var resource interface{}
		var err error
		if !out[0].IsNil() {
			resource = out[0].Interface()
		}
		if !out[1].IsNil() {
			err = out[1].Interface().(error)
		}
		return resource, err
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

const parameter = " parameter"
const musthave = "must specify a "
const maynothave = "may not have a "

type named_parameter struct {
	req    requirement
	name   string
	typ    reflect.Type
	toggle *bool
}

func (p named_parameter) FullName() string {
	return p.name + " " + p.typ.String()
}

// TODO: Unmentalize this function to make it more understandable
func analyseInputs(E error_f, ctx method_context, p parameter_spec) (method_spec, error) {
	m := method_spec{}
	if ctx.method_type.NumIn()-1 > p.maxParams() {
		return m, E("may accept at most", p.maxParams(), "parameters")
	}
	ordering := []int{}

	// This ordering is important. Only methods which have parameters
	// corrsponding to this ordering are valid. Others will generate
	// errors.
	var (
		parameter_parent_ids = named_parameter{p.ParentIDs, "parentIDs", reflect.MapOf(string_T, string_T), &m.uses_parent_ids}
		parameter_id         = named_parameter{p.ID, "id", string_T, &m.uses_id}
		parameter_payload    = named_parameter{p.Payload, "payload", ctx.owner_pointer_type, &m.uses_payload}
	)
	ordered_parameters := []named_parameter{
		parameter_parent_ids,
		parameter_id,
		parameter_payload,
	}

	if err := add_parameter_specs(E, &ordering, ctx, ordered_parameters); err != nil {
		return m, err
	}

	correct_order := []string{}
	if !parameter_order_correct(ordering) {
		if m.uses_parent_ids {
			correct_order = append(correct_order, parameter_parent_ids.FullName())
		}
		if m.uses_id {
			correct_order = append(correct_order, parameter_id.FullName())
		}
		if m.uses_payload {
			correct_order = append(correct_order, parameter_payload.FullName())
		}
		return m, E("Parameters out of order. Correct order is: (" + strings.Join(correct_order, ", ") + ")")
	}
	return m, nil
}

func add_parameter_specs(E error_f, ordering *[]int, ctx method_context, params []named_parameter) error {
	for _, p := range params {
		if _, err := add_parameter_spec(E, p.toggle, ordering, ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func add_parameter_spec(E error_f, toggle *bool, ordering *[]int, ctx method_context, p named_parameter) (bool, error) {
	if added, order, err := methodHasExactlyOneParameterOfType(E, ctx.method_type, p.typ); err != nil {
		return false, err
	} else if p.req.Required() && !added {
		return false, E(musthave, p.FullName(), parameter)
	} else if !added {
		return false, nil
	} else if !p.req.Allowed() {
		return false, E(maynothave, p.FullName(), parameter)
	} else {
		*ordering = append(*ordering, order)
		*toggle = true
		return true, nil
	}
}

func parameter_order_correct(ordering []int) bool {
	last := 0
	for _, o := range ordering {
		last = last + 1
		if o != last {
			return false
		}
	}
	return true
}

var (
	string_T = reflect.TypeOf("")
)

func methodHasExactlyOneParameterOfType(E error_f, method_type reflect.Type, parameter_type reflect.Type) (bool, int, error) {
	count := 0
	pos := -1
	for i := 0; i < method_type.NumIn(); i++ {
		if method_type.In(i) == parameter_type {
			count++
			pos = i
		}
	}
	if count > 1 {
		return false, -1, E("has multiple", parameter_type, "parameters")
	}
	return count == 1, pos, nil
}

func analyseOutputs(E error_f, ctx method_context) error {
	if ctx.method_type.NumOut() != 2 {
		return E("should have 2 outputs")
	} else if ctx.method_type.Out(0) != ctx.owner_pointer_type {
		return E("first output must be *" + ctx.owner_pointer_type.Elem().Name() + " (not " + ctx.method_type.Out(0).Name() + ")")
	} else if ctx.method_type.Out(1).Name() != "error" {
		return E("second output must be error (not " + ctx.method_type.Out(1).Name() + "")
	}
	return nil
}

func routeError(args ...interface{}) error {
	return Error("ROUTING: ", args)
}

func methodError(t reflect.Type, methodName string, args ...interface{}) error {
	prependage := t.Name() + "." + methodName
	args = append([]interface{}{prependage}, args...)
	return routeError(args...)
}

func newPtrTo(t reflect.Type) interface{} {
	return reflect.New(t).Interface()
}
