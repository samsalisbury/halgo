package halgo

import "reflect"

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

// This should only be called on the root node
func (n *node) Resolve(path ...string) (*resolved_node, error) {
	r := resolved_node{n, nil, ""}
	return r.Resolve(path...)
}

func (n *resolved_node) Resolve(path ...string) (*resolved_node, error) {
	if len(path) == 0 {
		return n, nil
	} else if c, ok := n.Child(path[0]); ok {
		node := &resolved_node{&c.node, n, path[0]}
		return node.Resolve(path[1:]...)
	}
	return nil, Error404(n.Path())
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
	node       node
	expansion  expansion
	child_type child_type
}

type child_type string

const (
	not_child     = child_type("none")
	child_pointer = child_type("pointer")
	child_map     = child_type("map")
	child_slice   = child_type("slice")
)

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
