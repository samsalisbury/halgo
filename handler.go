package halgo

import (
	"fmt"
	"net/http"
	"strings"
)

func NewServer(root Resource) (server, error) {
	if routes, err := newNode(root); err != nil {
		return server{}, err
	} else {
		return server{routes}, nil
	}
}

type server struct {
	routes node
}

func (s server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Print("request ", r.URL)
	if res, err := s.process(r); err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	} else {
		write(w, res)
	}
}

func write(w http.ResponseWriter, resource interface{}) {
	res := fmt.Sprint(resource)
	w.Write([]byte(res))
}

func (s server) process(r *http.Request) (*response, error) {
	path := strings.Split(r.URL.Path[1:], "/")
	Print(path)
	if n, err := resolve(s.routes, path, map[string]string{}); err != nil {
		return nil, err
	} else if m, ok := n.methods[r.Method]; !ok {
		return nil, Error405(r.Method, n)
	} else if prepared_request, err := prepare_request(r, n, m); err != nil {
		return nil, err
	} else {
		return invoke_method(n, m, prepared_request)
	}
}

type prepared_request struct {
	parentIds map[string]string
	id        string
	payload   interface{}
}

func prepare_request(r *http.Request, n resolved_node, m *method_info) (*prepared_request, error) {
	var (
		parentIds map[string]string
		id        string
		payload   interface{}
		err       error
	)
	if m.spec.uses_parent_ids {
		parentIds = n.route_values
	}
	if m.spec.uses_id {
		id = n.id
	}
	if m.spec.uses_payload {
		payload, err = prepare_payload(r.Body, m.ctx.owner_pointer_type)
	}
	return &prepared_request{parentIds, id, payload}, err
}

//type generic_http_method func(parentIDs map[string]string, id string, payload interface{}) (interface{}, error)
func invoke_method(n resolved_node, m *method_info, r *prepared_request) (*response, error) {
	if resource, err := m.method(r.parentIds, r.id, r.payload); err != nil {
		return nil, err
	} else {
		// TODO: Post-processing to add links
		return &response{200, resource, nil}, nil
	}
}

func resolve(n node, path []string, values map[string]string) (resolved_node, error) {
	Print("resolve", path)
	if len(path) == 1 {
		return resolved_node{n, path[0], values}, nil
	} else if len(path) == 0 {
		return resolved_node{n, "", values}, nil
	} else {
		return resolve_children(n.children, path, values)
	}
}

func resolve_children(r routes, path []string, values map[string]string) (resolved_node, error) {
	Print("resolve_children", path)
	if node, ok := r.child(path[0]); ok {
		return resolve(node, path[1:], values)
	} else {
		return resolved_node{}, Error404(path[0])
	}
}

func (r routes) child(name string) (node, bool) {
	if n, ok := r[name]; ok {
		return n, true
	}
	for _, n := range r {
		if n.is_identity {
			return n, true
		}
	}
	return node{}, false
}

type response struct {
	status int
	entity interface{}
	links  map[string]string
}

type resolved_route []resolved_node

type resolved_node struct {
	node
	id           string
	route_values map[string]string
}
