package halgo

import (
	"encoding/json"
	"io"
	"net/http"
)

func NewServer(root interface{}) (server, error) {
	if routes, err := buildGraph(root); err != nil {
		return server{}, err
	} else {
		return server{routes}, nil
	}
}

type server struct {
	routes *node
}

func (s server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Print("request ", r.URL)
	if response, err := s.process_request(r); err != nil {
		if httpError, ok := err.(HTTPError); ok {
			w.WriteHeader(httpError.StatusCode)
		} else {
			w.WriteHeader(500)
		}
		w.Write([]byte(err.Error()))
	} else if response == nil {
		w.WriteHeader(204)
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
	if node, err := s.routes.Resolve(r.URL.Path); err != nil {
		return nil, err
	} else if node == nil {
		return nil, Error404("(Node was nil.) " + r.URL.Path)
	} else if method, ok := node.BindMethod(r.Method); !ok {
		return nil, Error405(r.Method, node)
	} else if err := method.SetPayload(r.Body); err != nil {
		return nil, err
	} else if entity, err := method.Invoke(); err != nil {
		return nil, err
	} else if entity == nil {
		return nil, nil // This will cause a 204 No Content in the layer above
	} else if links, err := makeLinks(node); err != nil {
		return nil, err
	} else if resource, err := appendLinks(entity, links); err != nil {
		return nil, err
	} else if response, err := prepare_response(node, resource); err != nil {
		return nil, err
	} else {
		return response, nil
	}
}

func (m *bound_method) Invoke() (interface{}, error) {
	ids := m.node.RouteIDs()
	if entity, err := m.method(ids, m.node.url_value, m.payload); err != nil {
		return nil, err
	} else if entity == nil {
		return nil, nil
	} else {
		return entity, nil
	}
}

func makeLinks(n *resolved_node) (*map[string]interface{}, error) {
	return &map[string]interface{}{
		"self": map[string]string{"href": n.Path()},
	}, nil
}

func appendLinks(entity interface{}, links *map[string]interface{}) (*map[string]interface{}, error) {
	if resource, err := toMap(entity); err != nil {
		return nil, err
	} else if _, hasLinks := (*resource)["_links"]; hasLinks {
		return nil, Error("Resource already has field '_links'")
	} else {
		(*resource)["_links"] = links
		return resource, nil
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

type prepared_request struct {
	selfLink  string
	parentIds map[string]string
	id        string
	payload   interface{}
}

func prepare_response(n *resolved_node, resource *map[string]interface{}) (*response, error) {
	if resource == nil {
		return nil, Error("Resource was nil. Ought to have returned 204 No Content by now.")
	} else {
		append_embedded_resources(n, resource)
		// TODO: Allow other responses, e.g. 201 Created/ 202 Accepted etc.
		return &response{200, &resource, nil}, nil
	}
}

func append_embedded_resources(n *resolved_node, resource *map[string]interface{}) {
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
			(*resource)[name] = map[string]string{"error": err.Error()}
		} else {
			(*resource)[name] = entity
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
		return node.methods[GET].method(n.RouteIDs(), n.RouteID(), nil)
	}
}

func toMap(resource interface{}) (*map[string]interface{}, error) {
	var v map[string]interface{}
	if buf, err := json.Marshal(resource); err != nil {
		return nil, err
	} else if err := json.Unmarshal(buf, &v); err != nil {
		return nil, err
	} else {
		return &v, nil
	}
}

type response struct {
	status int
	entity interface{}
	links  map[string]string
}
