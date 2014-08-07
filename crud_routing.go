package halgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
)

var error_T = reflect.TypeOf((*error)(nil)).Elem()

func (root *Node) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path[1:], "/")
	if statusCode, entity, err := root.serve(path, r); err != nil {
		writeError(w, err)
	} else {
		w.WriteHeader(statusCode)
		if body, err := json.MarshalIndent(entity, "", "\t"); err != nil {
			panic("Unable to serialise entity: " + err.Error())
		} else {
			w.Write(body)
		}
	}
}

func writeError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case HTTPError:
		writeHttpError(w, e)
	default:
		w.WriteHeader(500)
		w.Write(serialise(err))
	}
}

func writeHttpError(w http.ResponseWriter, err HTTPError) {
	w.WriteHeader(err.StatusCode)
	w.Write(serialise(err.HalgoError))
}

func serialise(a interface{}) []byte {
	if data, err := json.Marshal(a); err != nil {
		panic(err)
	} else {
		return data
	}
}

func (root *Node) serve(path []string, r *http.Request) (int, interface{}, error) {
	if statusCode, entity, err := root.serveMainEntity(path, r); err != nil {
		return 0, nil, err
	} else {
		// TODO Manifest children
		return statusCode, entity, err
	}
}

func (root *Node) serveMainEntity(path []string, r *http.Request) (int, interface{}, error) {
	if target_node, parent_entity, id, err := root.Resolve(path, "", nil); err != nil {
		return 0, nil, err
	} else if method, ok := target_node.HTTPMethods[r.Method]; !ok {
		return 405, target_node.MethodNotSupportedBody(), nil
	} else if err != nil {
		return 0, nil, err
	} else {
		in := makeStdInputs(parent_entity, id, target_node, r)
		return method.Invoke(&target_node.Methods, in)
	}
}

func makeStdInputs(parent interface{}, id string, n *Node, r *http.Request) *StandardHTTPMethodInputs {
	in := &StandardHTTPMethodInputs{Node: n}
	if n.IsIdentity != nil && *n.IsIdentity {
		in.ID = id
	}
	if parent != nil {
		in.Parent = parent
	}
	return in
}

type StandardHTTPMethodInputs struct {
	Node   HttpNode
	Self   interface{}
	Parent interface{}
	ID     string
	Posted func() interface{}
}

type list []string

func (l *list) Add(s string) {
	(*l) = append(*l, s)
}

func (l *list) String() string {
	return strings.Join(*l, ", ")
}

func (n *Node) MethodNotSupportedBody() error {
	supported := &list{}
	if n.SupportsGET() {
		supported.Add("HEAD")
		supported.Add("GET")
	}
	if n.SupportsPUT() {
		supported.Add("PUT")
	}
	if n.SupportsPATCH() {
		supported.Add("PATCH")
	}
	if n.SupportsDELETE() {
		supported.Add("DELETE")
	}
	if n.SupportsPOST() {
		supported.Add("POST")
	}
	return Error("Supported Methods: ", supported)
}

func processPutResponse(statusCode int, entity interface{}, err error) *RESP {
	// TODO
	return nil
}

func processHeadResponse(statusCode int, entity interface{}, err error) *RESP {
	if data, err := json.MarshalIndent(entity, "", "\t"); err != nil {
		return InternalServerError("Unable to JSON serialise outgoing entity: " + err.Error())
	} else {
		return &RESP{
			StatusCode: statusCode,
			Body:       data,
		}
	}
}

type RESP struct {
	StatusCode int
	Body       []byte
}

func processGetResponse(statusCode int, entity interface{}, err error) *RESP {
	if data, err := json.MarshalIndent(entity, "", "\t"); err != nil {
		return InternalServerError("Unable to JSON serialise outgoing entity: " + err.Error())
	} else {
		return &RESP{
			StatusCode: statusCode,
			Body:       data,
		}
	}
}

func bytesBody(body []byte) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBuffer(body))
}

func stringBody(body string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBufferString(body))
}

func InternalServerError(message string) *RESP {
	r := &RESP{}
	r.StatusCode = 500
	r.Body = []byte(message)
	return r
}

func (n *Node) Resolve(path []string, id string, parent interface{}) (endpoint *Node, endpointParent interface{}, endpointID string, err error) {

	if len(path) == 0 || (len(path) == 1 && len(path[0]) == 0) {
		// This is either the end of the path, so return what we have.
		return n, parent, id, nil
	}

	// Try to find the next child.
	var child *Child
	if n.ID_Child != nil {
		child = n.ID_Child
	} else if c, ok := (*n.Children)[path[0]]; !ok {
		return n, nil, "", Error404(path[0])
	} else {
		child = c
	}

	// Now, we manifest the current node's entity, to use as the parent for
	// the next (child) node.
	if entity, err := n.Methods.Manifest(parent, id); err != nil {
		return nil, nil, "", err
	} else if entity == nil {
		// This node does not exist, so we can't move to the child.
		return nil, nil, "", nil
	} else {
		return child.Node.Resolve(path[1:], path[0], entity)
	}
}

func Graph(root interface{}) (HttpNode, error) {
	return graph(reflect.TypeOf(root), nil)
}

func graph(t reflect.Type, parent reflect.Type) (HttpNode, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Map {
		t = t.Elem()
	} else if t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	n := &Node{EntityType: t, EntityPtrType: reflect.PtrTo(t), ParentType: parent}
	if err := n.CompileMethods(); err != nil {
		return nil, err
	} else if err := n.AddChildren(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *Node) AddChildren() error {
	members := map[string]*Child{}
	collections := []*Child{}
	numFields := n.EntityType.NumField()
	for i := 0; i < numFields; i++ {
		f := n.EntityType.Field(i)
		if meta, err := getMetadata(f); err != nil {
			return err
		} else if meta.expansion != nil {
			if childNode, err := graph(f.Type, n.EntityPtrType); err != nil {
				return err
			} else {
				c := &Child{childNode.Node(), meta, f.Type.Kind()}
				if childNode.IsID() {
					collections = append(collections, c)
				} else {
					members[strings.ToLower(f.Name)] = c
				}
			}
		}
	}
	if len(collections) > 1 {
		return Error(n, "contains more than one collection child")
	} else if len(collections) == 1 && len(members) != 0 {
		return Error(n, "contains a collection child and named members")
	} else if len(collections) == 1 {
		n.ID_Child = collections[0]
	} else {
		n.Children = &members
	}
	return nil
}

type Node struct {
	EntityType    reflect.Type
	EntityPtrType reflect.Type
	Methods       compiled_methods
	ParentType    reflect.Type
	Children      *map[string]*Child
	IsIdentity    *bool
	ID_Child      *Child
	HTTPMethods   map[string]HTTPMethodDescriptor
}

type HttpNode interface {
	Node() *Node
	IsID() bool
	M() compiled_methods
	ServeHTTP(http.ResponseWriter, *http.Request)
}

func (n *Node) Node() *Node {
	return n
}

func (n *Node) M() compiled_methods {
	return n.Methods
}

func (n *Node) IsID() bool {
	if n.IsIdentity == nil {
		panic(n.EntityType.Name() + " had nil for IsIdentity.")
	} else {
		return *n.IsIdentity
	}
}

func (n *Node) SupportsGET() bool {
	return n.Methods.Manifest != nil
}

func (n *Node) SupportsPUT() bool {
	return n.Methods.Write != nil
}
func (n *Node) SupportsDELETE() bool {
	return n.Methods.Delete != nil
}

func (n *Node) SupportsPATCH() bool {
	return false
}

func (n *Node) SupportsPOST() bool {
	return false
}

type Child struct {
	Node *Node
	Meta meta
	Kind reflect.Kind
}

func (n *Node) AssertParentType(methodName string, parent reflect.Type) error {
	if n.ParentType == nil {
		return Error(n, " has no parent, but method ", methodName, " demands one.")
	} else if n.ParentType != parent {
		return Error(n.EntityType.Name(), " has parent type ", n.ParentType, " but method ", methodName, " asks for parent type ", parent)
	}
	return nil
}

func (n *Node) AssertIdentity(isIdentity bool) error {
	if n.IsIdentity == nil {
		n.IsIdentity = &isIdentity
	} else if isIdentity != n.IsID() {
		return Error(n.EntityType, " has inconsistent methods. Either all must accept a string parameter, or none.")
	}
	return nil
}

type compiled_methods struct {
	Exists   Exists_C
	Manifest Manifest_C
	Validate Validate_C
	Write    Write_C
	Delete   Delete_C
	Process  Process_C
}

// Exists may be specified to optimise the situation where manifesting a resource
// is more expensive than simply asserting that it exists.
// Params: parent, id, self, input
type Exists_C func(interface{}, string) (bool, error)

// Manifest == GET, also used for HEAD
// Params: parent, id
type Manifest_C func(interface{}, string) (interface{}, error)

// Params: parent, id, self
type Validate_C func(interface{}, string, interface{}) error

// Write is called for both PUT and POST (POSTs are converted to PUT-like operations internally)
// Params: parent, id, self
type Write_C func(interface{}, string, interface{}) error

// Guess which HTTP method Delete corresponds with...
// Params: parent, id, self
type Delete_C func(interface{}, string, interface{}) error

// Process == POST
// Params: parent, id, self, input
type Process_C func(interface{}, string, interface{}, interface{}) (interface{}, error)

//
// User method specs
//
type user_methods struct {
	Exists   Exists_U
	Manifest Manifest_U
	Validate Validate_U
	Write    Write_U
	Delete   Delete_U
	Process  Process_U
}

var user_methods_T = reflect.TypeOf(user_methods{})

type Exists_U func(interface{}, string) (bool, error)
type Manifest_U func(interface{}, string) error
type Validate_U func(interface{}, string) error
type Write_U func(interface{}, string) error
type Delete_U func(interface{}, string) error
type Process_U func(interface{}, string, interface{}) (interface{}, error)

func makeExists(s StandardMethod) Exists_C {
	return func(parent interface{}, id string) (bool, error) {
		out := s(&standardInputs{Parent: parent, ID: id})
		return *out.TrueOrFalse, out.Error
	}
}

func makeManifest(s StandardMethod) Manifest_C {
	return func(parent interface{}, id string) (interface{}, error) {
		out := s(&standardInputs{Parent: parent, ID: id})
		return out.Self, out.Error
	}
}

func makeValidate(s StandardMethod) Validate_C {
	return func(parent interface{}, id string, self interface{}) error {
		out := s(&standardInputs{Self: self, Parent: parent, ID: id})
		return out.Error
	}
}

func makeWrite(s StandardMethod) Write_C {
	return func(parent interface{}, id string, self interface{}) error {
		out := s(&standardInputs{Self: self, Parent: parent, ID: id})
		return out.Error
	}
}

func makeDelete(s StandardMethod) Delete_C {
	return func(parent interface{}, id string, self interface{}) error {
		out := s(&standardInputs{Self: self, Parent: parent, ID: id})
		return out.Error
	}
}

func makeProcess(s StandardMethod) Process_C {
	return func(parent interface{}, id string, self interface{}, otherIn interface{}) (otherOut interface{}, err error) {
		//_, _, otherOut, err = s(self, parent, id, otherIn)
		// TODO: Implement this
		panic("Process not implemented by the framework.")
		return otherOut, err
	}
}

func standardToCompiledMethod(name string, s StandardMethod) interface{} {
	switch name {
	case "Exists":
		return makeExists(s)
	case "Manifest":
		return makeManifest(s)
	case "Validate":
		return makeValidate(s)
	case "Write":
		return makeWrite(s)
	case "Delete":
		return makeDelete(s)
	case "Process":
		return makeProcess(s)
	default:
		panic("Compiled method " + name + " not defined.")
	}
}

var compiled_methods_T = reflect.TypeOf(compiled_methods{})

type standardised_methods struct {
	Exists   StandardMethod
	Manifest StandardMethod
	Validate StandardMethod
	Write    StandardMethod
	Delete   StandardMethod
	Process  StandardMethod
}

func (n *Node) CompileMethods() error {
	compiled := reflect.ValueOf(&compiled_methods{})
	numCompiled := compiled.Elem().NumField()
	for i := 0; i < numCompiled; i++ {
		name := compiled_methods_T.Field(i).Name
		if s, err := n.CompileMethod(name); err != nil {
			return err
		} else {
			standard := standardToCompiledMethod(name, s)
			compiled.Elem().FieldByName(name).Set(reflect.ValueOf(standard))
		}
	}
	// TODO: Refactor (patched with ugly indirection)
	n.Methods = *(compiled.Interface().(*compiled_methods))
	// Validate method set
	if n.Methods.Manifest == nil {
		return Error("*"+fmt.Sprint(n.EntityType), " does not have a Manifest method")
	}
	// Apply patched exists method if none provided
	if n.Methods.Exists == nil {
		n.Methods.Exists = convertManifestToExists(n.Methods.Manifest)
	}
	// Generate Basic set of HTTP methods
	n.HTTPMethods = map[string]HTTPMethodDescriptor{}
	for k, v := range methodTemplates {
		if v.IsSupported(&n.Methods) {
			n.HTTPMethods[k] = v
		}
	}
	return nil
}

func convertManifestToExists(m Manifest_C) Exists_C {
	return func(parent interface{}, id string) (bool, error) {
		entity, err := m(parent, id)
		return entity != nil, err
	}
}

func (n *Node) CompileMethod(name string) (StandardMethod, error) {
	if compiledMethod_F, ok := user_methods_T.FieldByName(name); !ok {
		panic("Compiled methods does not have a member named " + name)
	} else if userMethod_M, ok := n.EntityPtrType.MethodByName(name); !ok {
		return nil, nil
	} else {
		compiledMethod_T := compiledMethod_F.Type
		userMethod_T := userMethod_M.Type

		if inMaker, err := n.analyseInputs(name, compiledMethod_T, userMethod_T); err != nil {
			return nil, err
		} else if outMaker, err := n.analyseOutputs(name, compiledMethod_T, userMethod_T); err != nil {
			return nil, err
		} else {
			return n.makeStandardMethod(name, inMaker, outMaker), nil
		}
	}
}

type StandardMethod func(*standardInputs) *standardOutputs

type standardInputs struct {
	Self   interface{}
	Parent interface{}
	ID     string
	Posted func(reflect.Type) (interface{}, error)
}

type standardOutputs struct {
	Self        interface{}
	TrueOrFalse *bool
	OtherEntity interface{}
	Error       error
}

func (n *Node) makeStandardMethod(name string, inMaker *inputMaker, outMaker *outputMaker) StandardMethod {
	return func(i *standardInputs) *standardOutputs {
		if in, err := inMaker.makeInputs(i); err != nil {
			return &standardOutputs{Error: err}
		} else {
			if i.Self == nil {
				i.Self = reflect.New(n.EntityType).Interface()
			}
			method := reflect.ValueOf(i.Self).MethodByName(name)
			out := method.Call(in)
			return outMaker.makeOutputs(i.Self, out)
		}
	}
}

type inputMaker struct {
	ParentRequired     bool
	IdRequired         bool
	PostedBodyRequired bool
	PostedBodyType     reflect.Type
}

func (im *inputMaker) makeInputs(in *standardInputs) ([]reflect.Value, error) {
	inputs := []reflect.Value{}
	if im.ParentRequired {
		inputs = append(inputs, reflect.ValueOf(in.Parent))
	}
	if im.IdRequired {
		inputs = append(inputs, reflect.ValueOf(in.ID))
	}
	if im.PostedBodyRequired {
		if body, err := in.Posted(im.PostedBodyType); err != nil {
			return nil, err
		} else {
			inputs = append(inputs, reflect.ValueOf(body))
		}
	}
	return inputs, nil
}

func (n *Node) analyseInputs(methodName string, compiledMethod_T reflect.Type, userMethod_T reflect.Type) (*inputMaker, error) {
	im := &inputMaker{}
	// inSpec is the order and type of *allowed* inputs (parameters)
	_, specMaxIn := readMethodInputs(compiledMethod_T)
	// actualIn is the order and type of the actual inputs
	actualIn, actualNumIn := readMethodInputs(userMethod_T)

	// Skip the first input, it's the receiver (the entity itself)
	actualIn = actualIn[1:]
	actualNumIn = len(actualIn)

	// Validate Inputs
	if actualNumIn > specMaxIn {
		return nil, n.methodError(methodName, "should have at most", specMaxIn, "parameter(s).")
	}
	if actualNumIn < 2 {
		if err := n.AssertIdentity(false); err != nil {
			return nil, err
		}
	}
	for i, actualT := range actualIn {
		if i == 0 {
			im.ParentRequired = true
			if err := n.AssertParentType(methodName, actualT); err != nil {
				return nil, err
			}
		} else if i == 1 {
			im.IdRequired = true
			if err := n.AssertIdentity(true); err != nil {
				return nil, err
			}
		} else if i == 2 {
			im.PostedBodyRequired = true
			im.PostedBodyType = actualT
		}
	}
	return im, nil
}

type outputMaker struct {
	EntityRequired      bool
	TrueOrFalseRequired bool
	OtherEntityRequired bool
	ErrorRequired       bool
}

func (om *outputMaker) makeOutputs(receiver interface{}, outVals []reflect.Value) *standardOutputs {
	out := &standardOutputs{}
	out.Self = receiver
	i := 0
	if om.TrueOrFalseRequired {
		b := outVals[i].Bool()
		out.TrueOrFalse = &b
		i++
	}
	if om.OtherEntityRequired {
		out.OtherEntity = outVals[i].Interface()
		i++
	}
	if om.ErrorRequired {
		e := outVals[i]
		if !e.IsNil() {
			out.Error = e.Interface().(error)
		}
		i++
	}
	return out
}

func (n *Node) analyseOutputs(name string, expectedMethod_T reflect.Type, userMethod_T reflect.Type) (*outputMaker, error) {
	// outSpec is the order and type of *required* outputs, plus one
	// extra at the start for the entity itself.
	expectedOutSpec, expectedNumOut := readMethodOutputs(expectedMethod_T)

	// actualOut is the order and type of the actual outputs
	actualOut, actualNumOut := readMethodOutputs(userMethod_T)

	// Validate Outputs
	if actualNumOut != expectedNumOut {
		return nil, n.methodError(name, "should have", expectedNumOut, "outputs.")
	}
	om := &outputMaker{}
	gotEntity := false
	for i, expectedT := range expectedOutSpec {
		if actualOut[i] != expectedT {
			return nil, n.methodError(name, ": output", i, "should by of type", expectedT)
		}
		if !gotEntity && expectedT == n.EntityType {
			om.EntityRequired = true
			gotEntity = true
		} else if expectedT == error_T {
			om.ErrorRequired = true
		} else if expectedT.Kind() == reflect.Bool {
			om.TrueOrFalseRequired = true
		} else {
			om.OtherEntityRequired = true
		}
	}
	return om, nil
}

func (n *Node) methodError(name string, args ...interface{}) error {
	parts := list{}
	for _, a := range args {
		parts.Add(fmt.Sprint(a))
	}
	message := strings.Join(parts, " ")
	return Error("*" + n.EntityType.Name() + "." + name + " " + message)
}

func readMethodInputs(t reflect.Type) ([]reflect.Type, int) {
	numIn := t.NumIn()
	types := make([]reflect.Type, numIn)
	for i := 0; i < numIn; i++ {
		types[i] = t.In(i)
	}
	return types, numIn
}

func readMethodOutputs(t reflect.Type) ([]reflect.Type, int) {
	numOut := t.NumOut()
	types := make([]reflect.Type, numOut)
	for i := 0; i < numOut; i++ {
		types[i] = t.Out(i)
	}
	return types, numOut
}
