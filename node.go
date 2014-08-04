package halgo

import (
	"reflect"
	"strings"
)

type resolved_node struct {
	*node
	parent    *resolved_node
	url_value string
}

type node struct {
	url_name    string
	typ         reflect.Type
	is_identity bool
	methods     map[string]*method_info
	children    map[string]*child
	id_child    *child
}

func (n *resolved_node) Path() string {
	p := []string{}
	for n != nil {
		p = append([]string{n.RouteID()}, p...)
		n = n.parent
	}
	// Remember the root node will have RouteID = ''
	return strings.Join(p, "/")
}

func (n *resolved_node) RouteID() string {
	if n.url_value != "" {
		return n.url_value
	} else {
		return n.url_name
	}
}

func (n *resolved_node) RouteIDs() map[string]string {
	ids := map[string]string{}

	for n.parent != nil {
		if n.parent.is_identity {
			ids[n.parent.url_name] = n.parent.url_value
		}
		n = n.parent
	}
	return ids
}

// This should only be called on the root node
func (n *node) Resolve(uriPath string) (*resolved_node, error) {
	path := strings.Split(uriPath[1:], "/")
	r := resolved_node{n, nil, ""}
	return r.Resolve(path...)
}

func (n *resolved_node) Resolve(path ...string) (*resolved_node, error) {
	if len(path) == 0 || (len(path) == 1 && len(path[0]) == 0) {
		// it's the last point in the path, or the path ended with /, which we ignore
		// TODO: Redirect paths ending / to paths without?
		return n, nil
	} else if c, ok := n.Child(path[0]); ok {
		node := &resolved_node{c.node, n, path[0]}
		return node.Resolve(path[1:]...)
	}
	return nil, Error404(n.Path() + "/" + path[0])
}

func (n *resolved_node) Child(id string) (*child, bool) {
	if n.id_child != nil {
		return n.id_child, true
	} else if c, ok := n.children[id]; ok {
		return c, true
	} else {
		return nil, false
	}
}

type child struct {
	node *node
	meta meta
	kind reflect.Kind
}

func (n node) canHandle(method string) bool {
	switch method {
	case "GET", "HEAD":
		// This case is mainly to assert that HEAD is always possible if GET
		// is implemented. GET is mandatory, so GET and HEAD are always
		// supported.
		return true
	default:
		_, yes := n.methods[method]
		return yes
	}
}
