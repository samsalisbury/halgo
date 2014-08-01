package halgo

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func NewServer(root interface{}) (server, error) {
	if routes, err := buildGraph(root); err != nil {
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
	if response, err := s.process_request(r); err != nil {
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

func (s server) process_request(r *http.Request) (*response, error) {
	path := strings.Split(r.URL.Path[1:], "/")
	if node, err := s.routes.Resolve(path...); err != nil {
		return nil, err
	} else if node == nil {
		return nil, Error404(r.URL.Path)
	} else if method, ok := node.BindMethod(r.Method); !ok {
		return nil, Error405(r.Method, node)
	} else if err := method.SetPayload(r.Body); err != nil {
		return nil, err
	} else if entity, err := method.Invoke(); err != nil {
		return nil, err
	} else if response, err := prepare_response(node, entity); err != nil {
		return nil, err
	} else {
		return response, nil
	}
}

func (m *bound_method) Invoke() (interface{}, error) {
	ids := m.node.RouteIDs()
	if entity, err := m.method(ids, m.node.url_value, m.payload); err != nil {
		return nil, err
	} else {
		return entity, nil
	}
}

func (n *resolved_node) BindMethod(name string) (*bound_method, bool) {
	if m, ok := n.methods[name]; !ok {
		return nil, false
	} else {
		return &bound_method{m, n, nil}, true
	}
}

func (m *bound_method) SetPayload(body io.ReadCloser) error {
	if !m.spec.uses_payload {
		return nil
	}
	if payload, err := prepare_payload(body, m.ctx.owner_pointer_type); err != nil {
		return err
	} else {
		m.payload = payload
		return nil
	}
}

type bound_method struct {
	*method_info
	node    *resolved_node
	payload interface{}
}

func (n *resolved_node) RouteIDs() map[string]string {
	ids := map[string]string{}
	for n.parent != nil {
		if n.parent.is_identity {
			ids[n.parent.url_name] = n.parent.url_value
		}
	}
	return ids
}

// func prepare_request(path string, body io.ReadCloser, n resolved_node, m *method_info) (*prepared_request, error) {
// 	var (
// 		parentIds map[string]string
// 		id        string
// 		payload   interface{}
// 		err       error
// 	)
// 	if m.spec.uses_parent_ids {
// 		parentIds = n.route_values
// 	}
// 	if m.spec.uses_id {
// 		id = n.id
// 	}
// 	if m.spec.uses_payload {
// 		payload, err = prepare_payload(body, m.ctx.owner_pointer_type)
// 	}
// 	return &prepared_request{path, parentIds, id, payload}, err
// }

// func (s server) process(r *http.Request) (*response, error) {
// 	path := strings.Split(r.URL.Path[1:], "/")
// 	println("PATH:", strings.Join(path, "/"))
// 	if n, err := resolve_node(nil, s.routes, path[0], path[1:], map[string]string{}); err != nil {
// 		println("Not resolved", r.RequestURI)
// 		return nil, err
// 	} else if m, ok := n.methods[r.Method]; !ok {
// 		println("Not supported method", r.RequestURI, r.Method)
// 		return nil, Error405(r.Method, n)
// 	} else if prepared_request, err := prepare_request(r.URL.Path, r.Body, n, m); err != nil {
// 		println("Error preparing request")
// 		return nil, err
// 	} else if response, err := invoke_method(n, m, prepared_request); err != nil {
// 		return nil, err
// 	} else {
// 		return response, nil
// 	}
// }

type prepared_request struct {
	selfLink  string
	parentIds map[string]string
	id        string
	payload   interface{}
}

//type generic_http_method func(parentIDs map[string]string, id string, payload interface{}) (interface{}, error)
// func invoke_method(n resolved_node, m *method_info, r *prepared_request) (*response, error) {
// 	if resource, err := m.method(r.parentIds, r.id, r.payload); err != nil {
// 		return nil, err
// 	} else {
// 		return prepare_response(n, resource, r.selfLink)
// 	}
// }

func prepare_response(n *resolved_node, resource interface{}) (*response, error) {
	if resource == nil {
		return &response{404, nil, nil}, nil
	} else if m, err := toMap(resource); err != nil {
		return nil, err
	} else {
		append_embedded_resources(n, &m)
		return &response{200, &m, nil}, nil
	}
}

func append_embedded_resources(n *resolved_node, m *map[string]interface{}) {
	for _, c := range n.children {
		e := c.expansion
		name := c.node.url_name
		var err error = nil
		var entity interface{} = nil
		if e.isMap {
			entity, err = create_child_map(name, n)
		} else if e.isSlice {
			entity, err = create_child_slice(name, n)
		} else {
			entity, err = create_named_child(e.expansion_type, name, n)
		}
		if err != nil {
			(*m)[name] = map[string]string{"error": err.Error()}
		} else {
			(*m)[name] = entity
		}
	}
}

func create_named_child(et expansion_type, name string, n *resolved_node) (interface{}, error) {
	if et == href {
		return map[string]string{"_self": n.Path() + "/" + name}, nil
	} else if et == full {
		if r, err := n.Resolve(name); err != nil {
			return nil, err
		} else {
			method, _ := r.BindMethod("GET")
			return method.Invoke()
		}
	} else {
		return nil, Error("fields(...) filter not yet implemented.")
	}
}

func create_child_map(name string, n *resolved_node) (map[string]interface{}, error) {
	return nil, Error("Child maps not yet implemented.")
}

func create_child_slice(name string, n *resolved_node) ([]interface{}, error) {
	return nil, Error("Child slices not yet implemented.")
}

func get_sub_resource(n *resolved_node, name string) (interface{}, error) {
	if node, err := n.Resolve(name); err != nil {
		return nil, err
	} else {
		println("sub_node: ", node.url_name, ":", node.url_value)

		return node.methods[GET].method(n.RouteIDs(), n.RouteID(), nil)

		// if sub_request, err := prepare_request(n.Path()+"/"+name, nil, *sub_node, sub_node.methods[GET]); err != nil {
		// 	return map[string]string{"error": "Error preparing sub-request: " + err.Error()}
		// } else if sub_response, err := invoke_method(*sub_node, sub_node.methods[GET], sub_request); err != nil {
		// 	return map[string]string{"error": "Unable to get sub-resource: " + err.Error()}
		// } else {
		// 	return sub_response.entity
		// }
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

// func resolve_node(parent *resolved_node, n node, id string, path []string, values map[string]string) (resolved_node, error) {
// 	if id == "" && len(path) == 0 {
// 		// This is root or a path ending in /
// 		return resolved_node{n, id, values, parent}, nil
// 	}
// 	if child, ok := n.children.child(id); !ok {
// 		return resolved_node{}, Error404(id)
// 	} else if len(path) == 0 {
// 		return resolved_node{child, id, values, parent}, nil
// 	} else {
// 		values[child.name] = id
// 		parent = &resolved_node{child, id, values, parent}
// 		return resolve_node(parent, child, path[0], path[1:], values)
// 	}
// }

// func (r routes) child(name string) (node, bool) {
// 	if n, ok := r[name]; ok {
// 		return n, true
// 	}
// 	for _, n := range r {
// 		if n.is_identity {
// 			return n, true
// 		}
// 	}
// 	return node{}, false
// }

type response struct {
	status int
	entity interface{}
	links  map[string]string
}

func (n *resolved_node) Path() string {
	p := ""
	for n != nil {
		p = "/" + n.RouteID() + p
		n = n.parent
	}
	return p
}

func (n *resolved_node) RouteID() string {
	if n.url_value != "" {
		return n.url_value
	} else {
		return n.url_name
	}
}

// func (n *resolved_node) Resolve(childID string) (*resolved_node, bool) {
// 	println("resolved_node.Resolve(", childID, ")")
// 	if c, ok := n.children.child(childID); !ok {
// 		return nil, false
// 	} else {
// 		values := n.route_values
// 		values[n.name] = n.RouteID()
// 		return &resolved_node{
// 			c,
// 			childID,
// 			values,
// 			n,
// 		}, true
// 	}
// }
