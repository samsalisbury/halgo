package halgo

import (
	"reflect"
	"strings"
)

func buildGraph(root interface{}) (n node, err error) {
	t := reflect.TypeOf(root)
	return makeGraph(t, "")
}

func makeGraph(t reflect.Type, fieldName string) (n node, err error) {
	if methods, err := getMethods(t); err != nil {
		return n, err
	} else if is_identity, err := verifyMethods(fieldName, methods); err != nil {
		return n, err
	} else if children, id_child, err := makeChildren(t); err != nil {
		return n, err
	} else {
		return node{makeUrlName(fieldName, is_identity), t, is_identity, methods, children, id_child}, nil
	}
}

func makeUrlName(fieldName string, is_identity bool) string {
	url_name := strings.ToLower(fieldName)
	if is_identity {
		return "{id:" + url_name + "}"
	}
	return url_name
}

func makeChildren(t reflect.Type) (map[string]*child, *child, error) {
	children := map[string]*child{}
	identity_children := []*child{}
	n := t.NumField()
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if c, is_child, err := makeChild(f); err != nil {
			return nil, nil, err
		} else if !is_child {
			continue
		} else if c.node.is_identity {
			identity_children = append(identity_children, c)
		} else {
			children[f.Name] = c
		}
	}
	if len(identity_children) > 1 {
		return nil, nil, Error(t, "has multiple children which require an ID.")
	}
	if len(identity_children) == 1 {
		if len(children) != 0 {
			return nil, nil, Error(t, "has named children and a child which requires an ID.")
		} else {
			return nil, identity_children[0], nil
		}
	}
	return children, nil, nil
}

func makeChild(f reflect.StructField) (*child, bool, error) {
	var (
		childType  = not_child
		mapKeyKind reflect.Kind
		t          = f.Type
	)
	// Is the field a pointer to something?
	if t.Kind() == reflect.Ptr {
		childType = child_pointer
		t = t.Elem()
	}
	// We allow child resources to be pointers, maps, slices, and pointers
	// to maps/slices. Thus, after the pointer test, we keep resolving in
	// case it was the latter.
	if t.Kind() == reflect.Map {
		childType = child_map
		mapKeyKind = t.Key().Kind()
		t = t.Elem()
	} else if t.Kind() == reflect.Slice {
		childType = child_slice
		t = t.Elem()
	}
	if !hasNamedGetMethod(t) {
		// It doesn't look like a resource.
		// Is the field tagged up for halgo? If so, we will error if the
		// field is incompatible with being a child resource.
		is_tagged_as_child := f.Tag.Get("halgo") != ""
		var err error
		if is_tagged_as_child {
			err = Error(f.Name, " is tagged as a halgo child '", f.Tag.Get("halgo"), "' however ", t, " does not implement GET, and thus is an illegal resource.")
		}
		return nil, false, err
	} else if t.Kind() == reflect.Map && mapKeyKind != reflect.String {
		// TODO: This may be too restrictive, consider opting-in to being a
		// child resource rather than assuming all things which implement GET()
		// are necessarily intended to be child resources.
		// Also, in future, other kinds of key may be acceptable.
		return nil, false, Error(f.Name, " points to a resource type, but its key type is not string. Only string keys are allowed for child collections.")
	} else if expansion, err := getFieldExpansion(f); err != nil {
		return nil, false, err
	} else if node, err := makeGraph(t, f.Name); err != nil {
		return nil, false, err
	} else {
		return &child{node, *expansion, childType}, true, nil
	}
}

func verifyMethods(url_name string, methods map[string]*method_info) (is_identity bool, err error) {
	if len(methods) == 0 {
		return false, Error(url_name, " does not have any HTTP methods.")
	} else if _, ok := methods[GET]; !ok {
		return false, Error(url_name, " does not have a GET method.")
	} else {
		// TODO: Check if GET requires ID. If so, all other methods MUST require ID as well.
		// Error if any don't. Vice versa also true, if GET does not require ID, then all other
		// methods must not require ID.
		//
		// TODO: If node has any parents which are identities, then it MUST require
		// parentIDs map parameter.
		return methods[GET].spec.uses_id, nil
	}
}

// func getChildResources(t reflect.Type) (map[string]child_resource, error) {
// 	child_resources := map[string]child_resource{}
// 	for i := 0; i < t.NumField(); i++ {
// 		cr := child_resource{}
// 		f := t.Field(i)
// 		ft := f.Type
// 		if ft.Kind() == reflect.Ptr {
// 			ft = ft.Elem()
// 		}
// 		if ft.Kind() == reflect.Map || ft.Kind() == reflect.Slice {
// 			ft = ft.Elem()
// 		}
// 		cr.Type = ft
// 		if hasNamedGetMethod(cr.Type) {
// 			if expansion, err := getFieldExpansion(f); err != nil {
// 				return nil, err
// 			} else {
// 				cr.expansion = *expansion
// 				// TODO: Use field name instead of type for map id
// 				// TODO: Also do this with paths
// 				child_resources[strings.ToLower(cr.Type.Name())] = cr
// 			}
// 		}
// 	}
// 	return child_resources, nil
// }

// func buildRoutes(root interface{}) (n node, err error) {
// 	return newNode(child_resource{reflect.TypeOf(root), expansion{full, nil, false, false}})
// }

// func newNode(r child_resource) (n node, err error) {
// 	println("newNode:", r.Type.Name())
// 	if methods, err := getMethods(r.Type); err != nil {
// 		return n, err
// 	} else if len(methods) == 0 {
// 		return n, Error(r.Type, "does not have any HTTP methods")
// 	} else if child_resources, err := getChildResources(r.Type); err != nil {
// 		return n, err
// 	} else if children, err := newRoutes(child_resources); err != nil {
// 		return n, err
// 	} else {
// 		expansions := map[string]expansion{}
// 		for name, c := range child_resources {
// 			expansions[name] = c.expansion
// 		}
// 		n = node{methods, children, expansions, "", false}
// 		if yes, err := node_requires_id(n); err != nil {
// 			return n, err
// 		} else if yes {
// 			n.is_identity = true
// 		}
// 		return n, nil
// 	}
// }

// func newRoutes(children map[string]child_resource) (routes, error) {
// 	r := routes{}
// 	for _, c := range children {
// 		fullname := c.Type.Name()
// 		name := strings.ToLower(strings.TrimSuffix(fullname, "Resource"))
// 		if node, err := newNode(c); err != nil {
// 			return r, err
// 		} else if _, nameAlreadyExists := r[name]; nameAlreadyExists {
// 			return r, Errorf("Conflicting routes found for %v", name)
// 		} else {
// 			node.name = name
// 			r[name] = node
// 		}
// 	}
// 	return r, nil
// }
// func assertIsResource(t reflect.Type) error {
// 	if !hasNamedGetMethod(t) {
// 		return Error(t.Name(), "does not have a method named", GET)
// 	}
// 	_, err := analyseGetter(t)
// 	return err
// }

// func node_requires_id(n node) (bool, error) {
// 	really := []bool{}
// 	for _, n := range n.methods {
// 		really = append(really, n.spec.uses_id)
// 	}
// 	first_answer := really[0]
// 	for _, r := range really[1:] {
// 		if r != first_answer {
// 			return false, Error(n.name, "requires ID parameter in some methods but not all. This is illegal.")
// 		}
// 	}
// 	return first_answer, nil
// }
