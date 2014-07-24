package main

import (
	"fmt"
	"net/http"
	"strings"
)

func NewServer(root Resource) (server, error) {
	if routes, err := NewRoutes(root); err != nil {
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
	if node, err := resolve(s.routes, r.URL.Path[1:], map[string]string{}); err != nil {
		return nil, err
	} else {
		return invoke_method(r.Method, node)
	}
}

func invoke_method(method string, n resolved_node) (*response, error) {
	return nil, Error("Not implemented.")
}

func resolve(n node, path string, values map[string]string) (resolved_node, error) {
	if path == "" {
		return resolved_node{n, values}, nil
	} else {
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			return resolve_children(n.children, parts[0], parts[1], values)
		} else {
			return resolve_children(n.children, parts[0], "", values)
		}
	}
}

func resolve_children(r route, id string, path string, values map[string]string) (resolved_node, error) {
	if node, ok := r[id]; ok {
		values["x"] = id
		return resolve(node, path, values)
	}
	return resolved_node{}, Error404(id)
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
