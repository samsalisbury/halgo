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
	if node, err := resolve(s.routes, path, map[string]string{}); err != nil {
		return nil, err
	} else {
		return invoke_method(r.Method, node)
	}
}

func invoke_method(method string, n resolved_node) (*response, error) {
	return nil, Error("Not implemented.")
}

func resolve(n node, path []string, values map[string]string) (resolved_node, error) {
	if len(path) == 0 {
		return resolved_node{n, values}, nil
	} else {
		return resolve_children(n.children, path, values)
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

func resolve_children(r routes, path []string, values map[string]string) (resolved_node, error) {
	if node, ok := r.child(path[0]); ok {
		return resolve(node, path[1:], values)
	} else {
		return resolved_node{}, Error404(path[0])
	}
}

type response struct {
	status int
	entity interface{}
	links  map[string]string
}

type resolved_route []resolved_node

type resolved_node struct {
	node
	route_values map[string]string
}
