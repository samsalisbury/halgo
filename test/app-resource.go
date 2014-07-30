package test

type AppResource struct {
	Name     string
	Versions AppVersions
}

func (AppResource) GET(name string) (*AppResource, error) {
	if versions, ok := the_apps[name]; !ok {
		return nil, Error404(name)
	} else {
		return &AppResource{
			Name:     name,
			Versions: versions,
		}, nil
	}
}
