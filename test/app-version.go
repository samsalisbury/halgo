package test

type AppVersion struct {
	ID      string
	Name    string
	Version string
}

func (AppVersion) GET(parentIDs map[string]string, version string) (*AppVersion, error) {
	println("PARENT IDS:")
	for k, v := range parentIDs {
		println("\t", k, "=", v)
	}
	name := parentIDs["apps"]
	if app, ok := the_apps[name]; !ok {
		return nil, Error404(name)
	} else if ver, ok := app.Versions[version]; !ok {
		return nil, Error404(name + " v" + version)
	} else {
		return &ver, nil
	}
}
