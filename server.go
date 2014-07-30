package halgo

import (
	"encoding/json"
	"net/http"
	"strings"
)

func NewServer(root interface{}) (server, error) {
	if routes, err := buildRoutes(root); err != nil {
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
	} else if response, err := invoke_method(n, m, prepared_request); err != nil {
		return nil, err
	} else {
		return response, nil
	}
}

type prepared_request struct {
	selfLink  string
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
	return &prepared_request{r.URL.Path, parentIds, id, payload}, err
}

//type generic_http_method func(parentIDs map[string]string, id string, payload interface{}) (interface{}, error)
func invoke_method(n resolved_node, m *method_info, r *prepared_request) (*response, error) {
	if resource, err := m.method(r.parentIds, r.id, r.payload); err != nil {
		return nil, err
	} else {
		return prepare_response(resource, r.selfLink)
	}
}

func prepare_response(resource interface{}, selfLink string) (*response, error) {
	if resource == nil {
		return &response{404, nil, nil}, nil
	} else if m, err := toMap(resource); err != nil {
		return nil, err
	} else {
		m["_links"] = map[string]string{
			"self": selfLink,
		}
		return &response{200, m, nil}, nil
	}
}

func toMap(resource interface{}) (map[string]interface{}, error) {
	var v map[string]interface{}
	if buf, err := json.Marshal(resource); err != nil {
		return v, err
	} else if err := json.Unmarshal(buf, &v); err != nil {
		return v, err
	} else {
		return v, nil
	}
}

func resolve_node(n node, id string, path []string, values map[string]string) (resolved_node, error) {
	if id == "" && len(path) == 0 {
		// This is root or a path ending in /
		return resolved_node{n, id, values}, nil
	}
	if child, ok := n.children.child(id); !ok {
		return resolved_node{}, Error404(id)
	} else if len(path) == 0 {
		return resolved_node{child, id, values}, nil
	} else {
		values[child.name] = id
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
