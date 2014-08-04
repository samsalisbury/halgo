package halgo

import (
	"reflect"
	"strings"
)

func buildGraph(root interface{}) (n *node, err error) {
	t := reflect.TypeOf(root)
	return makeGraph(t, nil)
}

func makeGraph(t reflect.Type, f *reflect.StructField) (*node, error) {
	if methods, err := getMethods(t); err != nil {
		return nil, err
	} else if is_identity, err := verifyMethods(t, methods); err != nil {
		return nil, err
	} else if children, id_child, err := makeChildren(t); err != nil {
		return nil, err
	} else {
		name := makeUrlName(f, is_identity)
		return &node{name, t, is_identity, methods, children, id_child}, nil
	}
}

func makeUrlName(f *reflect.StructField, is_identity bool) string {
	if f == nil {
		return ""
	}
	if is_identity {
		fi := fieldInfo(*f)
		return strings.ToLower(fi.UnderlyingType.Name())
	}
	return strings.ToLower(f.Name)
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
			children[makeUrlName(&f, false)] = c
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

func fieldInfo(f reflect.StructField) field_info {
	t := f.Type
	k := t.Kind()
	// Is the field a pointer to something?
	if k == reflect.Ptr {
		t = t.Elem()
	}
	// We allow child resources to be pointers, maps, slices, and pointers
	// to maps/slices. Thus, after the pointer test, we keep resolving in
	// case it was the latter.
	if t.Kind() == reflect.Map {
		keyKind := t.Key().Kind()
		return field_info{t.Kind(), t.Elem(), &keyKind}
	} else if t.Kind() == reflect.Slice {
		return field_info{t.Kind(), t.Elem(), nil}
	} else {
		return field_info{k, t, nil}
	}
}

type field_info struct {
	Kind           reflect.Kind
	UnderlyingType reflect.Type
	KeyKind        *reflect.Kind
}

func makeChild(f reflect.StructField) (*child, bool, error) {
	fi := fieldInfo(f)
	if !hasNamedGetMethod(fi.UnderlyingType) {
		// It doesn't look like a resource.
		// Is the field tagged up for halgo? If so, we will error if the
		// field is incompatible with being a child resource.
		is_tagged_as_child := f.Tag.Get("halgo") != ""
		var err error
		if is_tagged_as_child {
			err = Error(f.Name, " is tagged as a halgo child '", f.Tag.Get("halgo"), "' however ", fi.UnderlyingType, " does not implement GET, and thus is an illegal resource.")
		}
		return nil, false, err
	} else if fi.Kind == reflect.Map && *fi.KeyKind != reflect.String {
		// TODO: This may be too restrictive, consider opting-in to being a
		// child resource rather than assuming all things which implement GET()
		// are necessarily intended to be child resources.
		// Also, in future, other kinds of key may be acceptable.
		return nil, false, Error(f.Name, " points to a resource type, but its key type is not string. Only string keys are allowed for child collections.")
	} else if metadata, err := getMetadata(f); err != nil {
		return nil, false, err
	} else if node, err := makeGraph(fi.UnderlyingType, &f); err != nil {
		return nil, false, err
	} else {
		return &child{node, metadata, fi.Kind}, true, nil
	}
}

func verifyMethods(t reflect.Type, methods map[string]*method_info) (is_identity bool, err error) {
	if len(methods) == 0 {
		return false, Error(t, " does not have any HTTP methods.")
	} else if _, ok := methods[GET]; !ok {
		return false, Error(t, " does not have a GET method.")
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
