package halgo

const (
	HEAD   = "HEAD"
	GET    = "GET"
	DELETE = "DELETE"
	PUT    = "PUT"
	PATCH  = "PATCH"
	POST   = "POST"
)

var parameter_specs map[string]parameter_spec = map[string]parameter_spec{
	HEAD:   parameter_spec{optional, optional, forbidden},
	GET:    parameter_spec{optional, optional, forbidden},
	DELETE: parameter_spec{required, optional, forbidden},
	PUT:    parameter_spec{required, optional, required},
	PATCH:  parameter_spec{required, optional, required},
	POST:   parameter_spec{optional, optional, required},
}

type requirement int

const (
	required  requirement = iota
	optional              = iota
	forbidden             = iota
)

func (r requirement) Allowed() bool {
	return r == required || r == optional
}

func (r requirement) Required() bool {
	return r == required
}

type parameter_spec struct {
	ID        requirement
	ParentIDs requirement
	Payload   requirement
}

func (p parameter_spec) maxParams() int {
	n := 0
	for _, r := range []requirement{p.ID, p.ParentIDs, p.Payload} {
		if r.Allowed() {
			n++
		}
	}
	return n
}
