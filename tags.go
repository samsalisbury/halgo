package halgo

import (
	"reflect"
	"strings"
)

type meta struct {
	expansion      *expansion
	child_link_rel *string
}

type link struct {
	rel  string
	href string
}

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

type tags map[string]map[string]string

func parseTags(f reflect.StructField) (tags, error) {
	data := f.Tag.Get("halgo")
	if data == "" {
		return nil, nil
	}
	tags := tags{}
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
			tags[parts[0]] = params
		}
	}
	return tags, nil
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

func getMetadata(f reflect.StructField) (m meta, err error) {
	if tags, err := parseTags(f); err != nil {
		return m, err
	} else if expansion, err := getFieldExpansion(tags, fieldInfo(f)); err != nil {
		return m, err
	} else if child_link_rel, err := getChildLinkRel(tags); err != nil {
		return m, err
	} else {
		return meta{expansion, child_link_rel}, nil
	}
}

func getChildLinkRel(t tags) (*string, error) {
	if l, ok := t["link"]; !ok {
		return nil, nil
	} else if rel, ok := l["rel"]; !ok {
		return nil, Error("Link must specify rel=<name>. E.g. link(rel=next).")
	} else {
		return &rel, nil
	}
}

func getFieldExpansion(t tags, fi field_info) (*expansion, error) {
	if embed, ok := t["embed"]; !ok {
		return nil, nil
	} else {
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
