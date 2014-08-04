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
	all    = expansion_type("all")
)

type expansion struct {
	expansion_type expansion_type
	fields         []string
	Field          field_info
}

type child_resource struct {
	Type      reflect.Type
	expansion expansion
}

// Tag format: `halgo:"thing1(param1=val1, param2=val2) thing2(param3, param4)"`
// e.g. `halgo:"embed(href)"`
//      `halgo:"link(rel=health)"`
//      `halgo:"embed(all) link(rel=apps)`
const whitespace_chars = " \t"

func parseTags(f reflect.StructField) (map[string]map[string]string, error) {
	data := f.Tag.Get("halgo")
	if data == "" {
		return nil, nil
	}
	commands := map[string]map[string]string{}
	for _, c := range strings.Split(data, " ") {
		c = strings.Trim(c, whitespace_chars)
		if len(c) == 0 {
			continue
		}
		parts := strings.Split(strings.TrimSuffix(c, ")"), "(")
		if !strings.HasSuffix(c, ")") || len(parts) != 2 {
			return nil, Error("Malformed halgo tag '", c, "'. Tags must be in the format 'name(...)'")
		}
		if params, err := parseTagParams(parts[1]); err != nil {
			return nil, err
		} else {
			commands[parts[0]] = params
		}
	}
	return commands, nil
}

func parseTagParams(p string) (map[string]string, error) {
	params := map[string]string{}
	for _, s := range strings.Split(p, ",") {
		parts := strings.Split(strings.Trim(s, whitespace_chars), "=")
		if len(parts) > 2 {
			return nil, Error("Tag parameters must be in the format paramName[=value]")
		} else if len(parts) == 2 {
			params[parts[0]] = parts[1]
		} else if len(parts) == 1 {
			params[parts[0]] = ""
		}
	}
	return params, nil
}

func getFieldExpansion(f reflect.StructField) (*expansion, error) {
	if tags, err := parseTags(f); err != nil {
		return nil, err
	} else if embed, ok := tags["embed"]; !ok {
		return nil, nil
	} else {
		fi := fieldInfo(f)
		if _, ok := embed[""]; ok {
			return &expansion{all, nil, fi}, nil
		}
		if _, ok := embed[string(all)]; ok {
			return &expansion{all, nil, fi}, nil
		}
		if _, ok := embed[string(href)]; ok {
			return &expansion{href, nil, fi}, nil
		}
		if _, ok := embed[string(fields)]; ok {
			// TODO: FIELDS!!!
			return &expansion{fields, nil, fi}, nil
		}
		return nil, Error("Embed type '", embed, " is not recognised.")
	}
}

// func getFieldExpansion(f reflect.StructField) (*expansion, error) {
// 	isMap, isSlice := false, false
// 	if f.Type.Kind() == reflect.Map {
// 		isMap = true
// 	} else if f.Type.Kind() == reflect.Slice {
// 		isSlice = true
// 	}
// 	if tag := f.Tag.Get("halgo"); tag == "" {
// 		return &expansion{none, nil, isMap, isSlice}, nil
// 	} else if !strings.HasPrefix(tag, "embed(") && !strings.HasPrefix(tag, "link(") {
// 		return nil, Error("Malformed halgo tag: ", tag, " (tags must begin with 'embed(')")
// 	} else {
// 		tag = strings.TrimPrefix(tag, "expand-")
// 		if tag == "none" {
// 			return &expansion{none, nil, isMap, isSlice}, nil
// 		} else if tag == "href" {
// 			return &expansion{href, nil, isMap, isSlice}, nil
// 		} else if tag == "full" {
// 			return &expansion{full, nil, isMap, isSlice}, nil
// 		} else if strings.HasPrefix(tag, "fields(") && strings.HasSuffix(tag, ")") {
// 			tag = strings.TrimSuffix(strings.TrimPrefix(tag, "fields("), ")")
// 			the_fields := strings.Split(tag, ",")
// 			for i, g := range the_fields {
// 				the_fields[i] = strings.Trim(g, " \t")
// 			}
// 			return &expansion{fields, the_fields, isMap, isSlice}, nil
// 		} else {
// 			return nil, Error("Malformed halgo tag: ", tag, " (expansion must be: 'none', 'href', 'full', or 'fields(comma, separated, fields)'' )")
// 		}
// 	}
// }
