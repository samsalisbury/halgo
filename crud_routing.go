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

func (n *Node) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path[1:], "/")
	println("Crawling", r.URL.Path, "length:", len(path))
	if endpoint, parent, id, err := n.crawl(path, "", nil); err != nil {
		println("Crawl /"+strings.Join(path, "/"), "failed:", err.Error())
		w.WriteHeader(500)
	} else {
		var statusCode int
		var entity interface{}
		//var err error
		switch r.Method {
		case HEAD:
			statusCode, entity, _ = endpoint.GET(nil, parent, id)
		case GET:
			statusCode, entity, _ = endpoint.GET(nil, parent, id)
		case PUT:
			if !endpoint.SupportsPUT() {
				statusCode, entity = n.MethodNotSupported()
			}
			// TODO: Parse the payload
			payload := (interface{})(nil)
			statusCode, entity, err = endpoint.PUT(payload, parent, id)
		default:
			// TODO: Prepare a 405 response
			entity = nil
		}

		w.WriteHeader(statusCode)
		if body, err := json.MarshalIndent(entity, "", "\t"); err != nil {
			panic("Unable to serialise entity: " + err.Error())
		} else {
			w.Write(body)
		}
	}
}

type list []string

func (l *list) Add(s string) {
	(*l) = append(*l, s)
}

func (l *list) String() string {
	return strings.Join(*l, ", ")
}

func (n *Node) MethodNotSupported() (int, error) {
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
	return 405, Error("Supported Methods: ", supported)
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

func (n *Node) crawl(path []string, id string, parent interface{}) (endpoint *Node, endpointParent interface{}, endpointID string, err error) {

	if len(path) == 0 || (len(path) == 1 && len(path[0]) == 0) {
		return n, parent, id, nil
	}
	var child *Child
	if n.ID_Child != nil {
		println("GOT ID CHILD: ", n.ID_Child.Node.EntityType.Name())
		child = n.ID_Child
	} else if c, ok := (*n.Children)[path[0]]; !ok {
		return n, nil, "", Error404(path[0])
	} else {
		child = c
	}

	if this_entity, err := n.Methods.Manifest(parent, id); err != nil {
		return nil, nil, "", err
	} else if this_entity == nil {
		return nil, nil, "", nil
	} else if _, err := child.Node.Methods.Manifest(this_entity, id); err != nil {
		return nil, nil, "", err
	} else {
		return child.Node.crawl(path[1:], path[0], this_entity)
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
}

type HttpNode interface {
	Node() *Node
	IsID() bool
	SupportsGET() bool
	SupportsPUT() bool
	SupportsDELETE() bool
	SupportsPATCH() bool
	SupportsPOST() bool
	ServeHTTP(http.ResponseWriter, *http.Request)
	GET(interface{}, interface{}, string) (int, interface{}, error)
	PUT(interface{}, interface{}, string) (int, interface{}, error)
	DELETE(interface{}, interface{}, string) (int, interface{}, error)
	//PATCH(interface{}, interface{}, string) (int, interface{}, error)
	//POST(interface{}, interface{}, string) (int, interface{}, error)
}

func (n *Node) Node() *Node {
	return n
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

func (n *Node) GET(_ interface{}, parent interface{}, id string) (int, interface{}, error) {
	if entity, err := n.Methods.Manifest(parent, id); err != nil {
		return 500, nil, err
	} else if entity == nil {
		return 404, nil, nil
	} else {
		return 200, entity, nil
	}
}

func (n *Node) SupportsPUT() bool {
	return n.Methods.Write != nil
}

func (n *Node) PUT(payload interface{}, parent interface{}, id string) (int, interface{}, error) {
	if exists, err := n.Methods.Exists(parent, id); err != nil {
		return 500, nil, err
	} else {
		status := 201
		if exists {
			status = 200
		}
		if err := n.Methods.Write(payload, id, parent); err != nil {
			return 500, nil, err
		}
		return status, payload, nil
	}
}

func (n *Node) SupportsDELETE() bool {
	return n.Methods.Delete != nil
}

func (n *Node) DELETE(null interface{}, parent interface{}, id string) (int, interface{}, error) {
	if exists, err := n.Methods.Exists(parent, id); err != nil {
		return 500, nil, err
	} else if !exists {
		return 404, nil, nil
	} else if err := n.Methods.Delete(null, id, parent); err != nil {
		return 500, nil, err
	} else {
		return 200, nil, nil
	}
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

func (n *Node) DeclareParentType(method string, parent reflect.Type) error {
	if n.ParentType == nil {
		return Error(n, " has no parent, but method ", method, " demands one.")
	} else if n.ParentType != parent {
		return Error(n.EntityType.Name(), " has parent type ", n.ParentType, " but method ", method, " asks for parent type ", parent)
	}
	return nil
}

func (n *Node) DeclareIdentity(isIdentity bool) error {
	if n.IsIdentity == nil {
		n.IsIdentity = &isIdentity
	} else if isIdentity != n.IsID() {
		return Error(n.EntityType, "has inconsistent methods. Either all must accept a string parameter, or none.")
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

func (n *Node) CompileMethods() error {
	n.Methods = compiled_methods{}
	if method, ok := n.EntityPtrType.MethodByName("Exists"); ok {
		if m, err := n.CompileExistsMethod(method); err != nil {
			return err
		} else {
			n.Methods.Exists = m
		}
	}
	if method, ok := n.EntityPtrType.MethodByName("Manifest"); ok {
		if m, err := n.CompileManifestMethod(method); err != nil {
			return err
		} else {
			n.Methods.Manifest = m
		}
	}
	// Validate method set
	if n.Methods.Manifest == nil {
		return Error("*"+fmt.Sprint(n.EntityType), " does not have a Manifest method")
	}
	// Apply patched exists method if none provided
	if n.Methods.Exists == nil {
		n.Methods.Exists = func(parent interface{}, id string) (bool, error) {
			entity, err := n.Methods.Manifest(parent, id)
			return entity != nil, err
		}
	}
	return nil
}

func (n *Node) CompileExistsMethod(method reflect.Method) (Exists_C, error) {
	if err := n.ValidateExistsMethod(method); err != nil {
		return nil, err
	}
	// These are bound once, outside the function, since Exists
	// always starts with nil and doesn't write to receiver
	receiver := reflect.New(n.EntityType)
	zeroBoundFunc := receiver.MethodByName("Exists")
	numIn := method.Type.NumIn()
	f := func(parent interface{}, id string) (bool, error) {
		in := makeInParams(parent, id, numIn)
		out := zeroBoundFunc.Call(in)
		var err error
		if !out[1].IsNil() {
			err = out[1].Interface().(error)
		}
		return out[0].Bool(), err
	}
	return f, nil
}

func makeInParams(parent interface{}, id string, numIn int) []reflect.Value {
	numIn--
	in := make([]reflect.Value, numIn)
	if numIn > 0 {
		in[0] = reflect.ValueOf(parent)
	}
	if numIn > 1 {
		in[1] = reflect.ValueOf(id)
	}
	return in
}

func (n *Node) CompileManifestMethod(method reflect.Method) (Manifest_C, error) {
	if err := n.ValidateManifestMethod(method); err != nil {
		return nil, err
	}
	numIn := method.Type.NumIn()
	f := func(parent interface{}, id string) (interface{}, error) {
		// println("Executing " + n.EntityType.Name() + ".Manifest(" + fmt.Sprint(parent) + ", '" + id + "')")
		// println("...which has", numIn, "inputs")
		// println("...of which the first is... ", method.Type.In(0).Elem().Name())
		receiver := reflect.New(n.EntityType)
		method := receiver.MethodByName("Manifest")
		in := makeInParams(parent, id, numIn)
		//println("...and we got", len(in), "inputs")
		out := method.Call(in)
		var entity interface{}
		var err error
		if !out[0].IsNil() {
			err = out[0].Interface().(error)
		}
		if !receiver.IsNil() {
			entity = receiver.Interface()
		}
		return entity, err
	}
	return f, nil
}

// All methods have the same input requirements. They all take either
// their parent, or that plus a string (i.e. ID)
func (n *Node) ValidateInputs(methodName string, method reflect.Method) error {
	//println(n.EntityType.Name()+"."+methodName+" has ", method.Type.NumIn(), " params")
	// Ignore the receiver for now.
	numIn := method.Type.NumIn() - 1
	if n.ParentType == nil {
		if numIn != 0 {
			// This is the root node, no inputs allowed
			return Error("Root node '", n, "' should not have any inputs on its ", methodName, " method.")
		}
		if err := n.DeclareIdentity(false); err != nil {
			return err
		}
		return nil
	}
	if numIn == 0 {
		if err := n.DeclareIdentity(false); err != nil {
			return err
		}
		return nil // Objection! Overruled. I'm going to allow this.
	}
	if numIn != 1 && numIn != 2 {
		return Error(n.EntityType, ".", methodName, " should have 1 or 2 input parameters")
	}
	if err := n.DeclareParentType(methodName, method.Type.In(1)); err != nil {
		return err
	}
	if numIn == 2 {
		if method.Type.In(2).Kind() != reflect.String {
			return Error(n.EntityType, "")
		} else if err := n.DeclareIdentity(true); err != nil {
			return err
		}
	} else if err := n.DeclareIdentity(false); err != nil {
		return err
	}
	return nil
}

func (n *Node) ValidateManifestMethod(method reflect.Method) error {
	if err := n.ValidateInputs("Manifest", method); err != nil {
		return err
	}
	detached := method.Func.Type()
	numOut := detached.NumOut()
	if numOut != 1 {
		return Error(n.EntityType, ".Manifest should have 1 output parameter")
	}
	out0 := detached.Out(0)
	if !out0.Implements(error_T) {
		return Error(n.EntityType, ".Manifest output parameter should implement error.")
	}
	return nil
}

func (n *Node) ValidateExistsMethod(method reflect.Method) error {
	if err := n.ValidateInputs("Exists", method); err != nil {
		return err
	}
	detached := method.Func.Type()
	numOut := detached.NumOut()
	if numOut != 2 {
		return Error(n.EntityType, ".Exists should have 2 output parameters")
	}
	out1 := detached.Out(0)
	if out1.Kind() != reflect.Bool {
		return Error(n.EntityType, ".Exists first output parameter should be bool")
	}
	out2 := detached.Out(1)
	if out2.Implements(error_T) {
		return Error(n.EntityType, ".Exists second output parameter should implement error.")
	}
	return nil
}
