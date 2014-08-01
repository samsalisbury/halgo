package halgo

import (
	"reflect"
	"strings"
)

type expansion_type string

const (
	none   = expansion_type("none")
	href   = expansion_type("href")
	fields = expansion_type("fields")
	full   = expansion_type("full")
)

type expansion struct {
	expansion_type expansion_type
	fields         []string
	isMap          bool
	isSlice        bool
}

type child_resource struct {
	Type      reflect.Type
	expansion expansion
}

func getFieldExpansion(f reflect.StructField) (*expansion, error) {
	isMap, isSlice := false, false
	if f.Type.Kind() == reflect.Map {
		isMap = true
	} else if f.Type.Kind() == reflect.Slice {
		isSlice = true
	}
	if tag := f.Tag.Get("halgo"); tag == "" {
		return &expansion{href, nil, isMap, isSlice}, nil
	} else if !strings.HasPrefix(tag, "expand-") {
		return nil, Error("Malformed halgo tag: ", tag, " (tags must begin with 'expand-')")
	} else {
		tag = strings.TrimPrefix(tag, "expand-")
		if tag == "none" {
			return &expansion{none, nil, isMap, isSlice}, nil
		} else if tag == "href" {
			return &expansion{href, nil, isMap, isSlice}, nil
		} else if tag == "full" {
			return &expansion{full, nil, isMap, isSlice}, nil
		} else if strings.HasPrefix(tag, "fields(") && strings.HasSuffix(tag, ")") {
			tag = strings.TrimSuffix(strings.TrimPrefix(tag, "fields("), ")")
			the_fields := strings.Split(tag, ",")
			for i, g := range the_fields {
				the_fields[i] = strings.Trim(g, " \t")
			}
			return &expansion{fields, the_fields, isMap, isSlice}, nil
		} else {
			return nil, Error("Malformed halgo tag: ", tag, " (expansion must be: 'none', 'href', 'full', or 'fields(comma, separated, fields)'' )")
		}
	}
}
