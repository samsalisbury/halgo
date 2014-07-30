package halgo

type RootResource struct {
	Welcome string  `json:"welcome"`
	Version string  `json:"version"`
	Apps    *Apps   `json:"apps" halgo:"expand-href"`
	Health  *Health `json:"-"`
}

func (RootResource) GET() (*RootResource, error) {
	println("Root handler")
	a := Apps{}
	if aa, err := a.GET(); err != nil {
		return nil, err
	} else {
		return &RootResource{
			Welcome: "Welcome to the deployment service API",
			Version: "0.0.110",
			Apps:    aa,
		}, nil
	}
}

type Health struct {
	Hello string
}

func (Health) GET() (*Health, error) {
	return &Health{
		Hello: "Feelin' good!",
	}, nil
}

type Apps struct {
	NumberOfApps int            `json:"numberOfApps"`
	Apps         map[string]App `json:"apps" halgo:"expand-full"`
}

func (Apps) GET() (*Apps, error) {
	return &Apps{
		NumberOfApps: len(the_apps),
		Apps:         the_apps,
	}, nil
}

type App struct {
	Name     string                `json:"name" halgo:"id"`
	Versions map[string]AppVersion `json:"versions" halgo:"expand-href"`
}

func (App) GET(name string) (*App, error) {
	println("App.GET(", name, ")")
	if app, ok := the_apps[name]; !ok {
		return nil, Error404(name)
	} else {
		return &App{
			Name:     name,
			Versions: app.Versions,
		}, nil
	}
}

var the_apps = map[string]App{
	"test-app": App{
		Name: "test-app",
		Versions: map[string]AppVersion{
			"0.2.0": AppVersion{"test-app-0-2-0", "test-app", "0.2.0"},
			"1.2.3": AppVersion{"test-app-1-2-3", "test-app", "1.2.3"},
			"1.3.9": AppVersion{"test-app-1-3-9", "test-app", "1.3.9"},
		},
	},
}

type AppVersion struct {
	ID      string `json:"id" halgo:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (AppVersion) GET(parentIDs map[string]string, version string) (*AppVersion, error) {
	println("PARENT IDS:")
	for k, v := range parentIDs {
		println("\t", k, "=", v)
	}
	name := parentIDs["app"]
	if app, ok := the_apps[name]; !ok {
		return nil, Error404(name)
	} else if ver, ok := app.Versions[version]; !ok {
		return nil, Error404(name + " v" + version)
	} else {
		return &ver, nil
	}
}
