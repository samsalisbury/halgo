package halgo

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	HEAD   = "HEAD"
	GET    = "GET"
	DELETE = "DELETE"
	PUT    = "PUT"
	PATCH  = "PATCH"
	POST   = "POST"
)

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
	HEAD:   parameter_spec{optional, optional, forbidden},
	GET:    parameter_spec{optional, optional, forbidden},
	DELETE: parameter_spec{required, optional, forbidden},
	PUT:    parameter_spec{required, optional, required},
	PATCH:  parameter_spec{required, optional, required},
	POST:   parameter_spec{optional, optional, required},
}

type routes map[string]node

type node struct {
	methods     map[string]*method_info
	children    routes
	expansions  map[string]expansion
	name        string
	is_identity bool
}

func (n node) SupportsMethod(m string) bool {
	if _, yes := n.methods[m]; yes {
		return true
	}
	return false
}

func buildRoutes(root interface{}) (n node, err error) {
	return newNode(child_resource{reflect.TypeOf(root), expansion{full, nil, false, false}})
}

func newNode(r child_resource) (n node, err error) {
	println("newNode:", r.Type.Name())
	if methods, err := getMethods(r.Type); err != nil {
		return n, err
	} else if len(methods) == 0 {
		return n, Error(r.Type, "does not have any HTTP methods")
	} else if child_resources, err := getChildResources(r.Type); err != nil {
		return n, err
	} else if children, err := newRoutes(child_resources); err != nil {
		return n, err
	} else {
		expansions := map[string]expansion{}
		for name, c := range child_resources {
			expansions[name] = c.expansion
		}
		n = node{methods, children, expansions, "", false}
		if yes, err := node_requires_id(n); err != nil {
			return n, err
		} else if yes {
			n.is_identity = true
		}
		return n, nil
	}
}

func newRoutes(children map[string]child_resource) (routes, error) {
	r := routes{}
	for _, c := range children {
		fullname := c.Type.Name()
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

type expansion_type string

const (
	none   = expansion_type("none")
	href   = expansion_type("href")
	fields = expansion_type("fields")
	full   = expansion_type("full")
)

type expansion struct {
	expansion_type expansion_type
	fields         []string
	isMap          bool
	isSlice        bool
}

type child_resource struct {
	Type      reflect.Type
	expansion expansion
}

func getChildResources(t reflect.Type) (map[string]child_resource, error) {
	child_resources := map[string]child_resource{}
	for i := 0; i < t.NumField(); i++ {
		cr := child_resource{}
		f := t.Field(i)
		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Map || ft.Kind() == reflect.Slice {
			ft = ft.Elem()
		}
		cr.Type = ft
		if hasNamedGetMethod(cr.Type) {
			if expansion, err := getFieldExpansion(f); err != nil {
				return nil, err
			} else {
				cr.expansion = *expansion
				// TODO: Use field name instead of type for map id
				// TODO: Also do this with paths
				child_resources[strings.ToLower(cr.Type.Name())] = cr
			}
		}
	}
	return child_resources, nil
}

func getFieldExpansion(f reflect.StructField) (*expansion, error) {
	isMap, isSlice := false, false
	if f.Type.Kind() == reflect.Map {
		isMap = true
	} else if f.Type.Kind() == reflect.Slice {
		isSlice = true
	}
	if tag := f.Tag.Get("halgo"); tag == "" {
		return &expansion{href, nil, isMap, isSlice}, nil
	} else if !strings.HasPrefix(tag, "expand-") {
		return nil, Error("Malformed halgo tag: ", tag, " (tags must begin with 'expand-')")
	} else {
		tag = strings.TrimPrefix(tag, "expand-")
		if tag == "none" {
			return &expansion{none, nil, isMap, isSlice}, nil
		} else if tag == "href" {
			return &expansion{href, nil, isMap, isSlice}, nil
		} else if tag == "full" {
			return &expansion{full, nil, isMap, isSlice}, nil
		} else if strings.HasPrefix(tag, "fields(") && strings.HasSuffix(tag, ")") {
			tag = strings.TrimSuffix(strings.TrimPrefix(tag, "fields("), ")")
			the_fields := strings.Split(tag, ",")
			for i, g := range the_fields {
				the_fields[i] = strings.Trim(g, " \t")
			}
			return &expansion{fields, the_fields, isMap, isSlice}, nil
		} else {
			return nil, Error("Malformed halgo tag: ", tag, " (expansion must be: 'none', 'href', 'full', or 'fields(comma, separated, fields)'' )")
		}
	}
}

func hasNamedGetMethod(t reflect.Type) bool {
	_, exists := t.MethodByName(GET)
	return exists
}

func assertIsResource(t reflect.Type) error {
	if !hasNamedGetMethod(t) {
		return Error(t.Name(), "does not have a method named", GET)
	}
	_, err := analyseGetter(t)
	return err
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
			owner_type:         reflect.TypeOf(instance).Elem(),
			owner_pointer_type: reflect.TypeOf(instance),
			bound_method:       reflect.ValueOf(instance).MethodByName(name),
			method_type:        method.Type,
		}, true
	} else {
		return method_context{}, false
	}
}

type method_context struct {
	owner_type         reflect.Type
	owner_pointer_type reflect.Type
	bound_method       reflect.Value
	method_type        reflect.Type
}

type error_f func(args ...interface{}) error

func analyseGetter(t reflect.Type) (m *method_info, err error) {
	E := func(args ...interface{}) error { return methodError(t, GET, args...) }
	if ctx, ok := analyseMethodContext(t, GET); !ok {
		return m, nil
	} else if err := analyseOutputs(E, ctx); err != nil {
		return m, err
	} else if method_spec, err := analyseInputs(E, ctx, parameter_specs[GET]); err != nil {
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
			in[i] = reflect.ValueOf(parentIDs)
			i++
		}
		if spec.uses_id {
			in[i] = reflect.ValueOf(id)
			i++
		}
		if spec.uses_payload {
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
	} else {
		out1_t := ctx.method_type.Out(0)
		out2_t := ctx.method_type.Out(1)
		if out1_t != ctx.owner_pointer_type {
			const format = "first output must be *%v (not %v)"
			message := fmt.Sprintf(format, ctx.owner_type, out1_t)
			return E(message)
			// return E("first output must be *" + ctx.owner_type.Name() + " (not " + fmt.Sprint(out1_t) + ")")
		} else if out2_t.Name() != "error" {
			return E("second output must be error (not " + fmt.Sprint(out2_t) + "")
		}
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
