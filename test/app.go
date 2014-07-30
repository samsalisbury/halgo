package test

type App struct {
	Name     string
	Versions map[string]AppVersion
}

func (App) HandleGET(name string) (*App, error) {
	if versions, ok := the_apps[name]; !ok {
		return nil, Error404(name)
	} else {
		return &App{
			Name:     name,
			Versions: versions,
		}, nil
	}
}
