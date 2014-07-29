package halgo

import (
	"encoding/json"
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
	if response, err := s.process(r); err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	} else {
		write(w, response)
	}
}

func write(w http.ResponseWriter, r *response) {
	if buf, err := json.MarshalIndent(r.entity, "", "\t"); err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(r.status)
		w.Write(buf)
	}
}

func (s server) process(r *http.Request) (*response, error) {
	path := strings.Split(r.URL.Path[1:], "/")
	println("PATH:", strings.Join(path, "/"))
	if n, err := resolve_node(s.routes, path[0], path[1:], map[string]string{}); err != nil {
		println("Not resolved", r.RequestURI)
		return nil, err
	} else if m, ok := n.methods[r.Method]; !ok {
		println("Not supported method", r.RequestURI, r.Method)
		return nil, Error405(r.Method, n)
	} else if prepared_request, err := prepare_request(r, n, m); err != nil {
		println("Error preparing request")
		return nil, err
	} else {
		println("Invoking method", m.ctx.owner_pointer_type.Elem().Name(), r.Method)
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

func resolve_node(n node, id string, path []string, values map[string]string) (resolved_node, error) {
	if id == "" && len(path) == 0 {
		// This is root or a path ending in /
		return resolved_node{n, id, values}, nil
	}
	values[n.name] = id
	if child, ok := n.children.child(id); !ok {
		return resolved_node{}, Error404(id)
	} else if len(path) == 0 {
		return resolved_node{child, id, values}, nil
	} else {
		return resolve_node(child, path[0], path[1:], values)
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

type resolved_node struct {
	node
	id           string
	route_values map[string]string
}
