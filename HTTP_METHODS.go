package halgo

type HTTPMethodDescriptor struct {
	IsSupported          func(m *compiled_methods) bool
	Invoke               StandardHTTPMethod
	RequiresEntity       bool
	RequiresOtherPayload bool
}

var methodTemplates = map[string]HTTPMethodDescriptor{
	GET:    GET_desc,
	DELETE: DELETE_desc,
	PUT:    PUT_desc,
}

// in: node, self, parent, id, otherIn; out: statusCode, entity, error
type StandardHTTPMethod func(*compiled_methods, *StandardHTTPMethodInputs) (int, interface{}, error)

var GET_desc = HTTPMethodDescriptor{
	IsSupported: func(m *compiled_methods) bool {
		return m.Manifest != nil
	},
	Invoke: func(m *compiled_methods, in *StandardHTTPMethodInputs) (int, interface{}, error) {
		if entity, err := m.Manifest(in.Parent, in.ID); err != nil {
			return 500, nil, err
		} else if entity == nil {
			return 404, nil, nil
		} else {
			return 200, entity, nil
		}
	},
}

var DELETE_desc = HTTPMethodDescriptor{
	IsSupported: func(m *compiled_methods) bool {
		return m.Delete != nil && m.Exists != nil
	},
	Invoke: func(m *compiled_methods, in *StandardHTTPMethodInputs) (int, interface{}, error) {
		if exists, err := m.Exists(in.Parent, in.ID); err != nil {
			return 500, nil, err
		} else if !exists {
			return 404, nil, nil
		} else if err := m.Delete(nil, in.ID, in.Parent); err != nil {
			return 500, nil, err
		} else {
			return 200, nil, nil
		}
	},
}

var PUT_desc = HTTPMethodDescriptor{
	IsSupported: func(m *compiled_methods) bool {
		return m.Write != nil
	},
	Invoke: func(m *compiled_methods, in *StandardHTTPMethodInputs) (int, interface{}, error) {
		if exists, err := m.Exists(in.Parent, in.ID); err != nil {
			return 500, nil, err
		} else {
			successStatus := 201
			if exists {
				successStatus = 200
			}
			if err := m.Write(in.Self, in.ID, in.Parent); err != nil {
				return 500, nil, err
			}
			return successStatus, in.Self, nil
		}
	},
	RequiresEntity: true,
}
