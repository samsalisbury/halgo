package halgo

type RootResource struct {
	Welcome string
	Version string
	Apps    *AppsResource
	Health  *HealthResource
}

func (RootResource) HandleGET() (*RootResource, error) {
	println("Root handler")
	return &RootResource{
		Welcome: "Welcome to the deployment service API",
		Version: "0.0.110",
	}, nil
}

type HealthResource struct {
	Hello string
}

func (HealthResource) HandleGET() (*HealthResource, error) {
	return &HealthResource{
		Hello: "Feelin' good!",
	}, nil
}

type AppsResource struct {
	NumberOfApps int
	AppVersions  map[string]AppVersionsResource
}

func (AppsResource) HandleGET() (*AppsResource, error) {
	return &AppsResource{
		NumberOfApps: len(the_apps),
		AppVersions:  the_apps,
	}, nil
}

type AppVersionsResource struct {
	Name string
	Apps map[string]AppResource
}

func (AppVersionsResource) HandleGET(name string) (*AppVersionsResource, error) {
	if appsResource, ok := the_apps[name]; !ok {
		return nil, Error404(name)
	} else {
		return &AppVersionsResource{
			Name: name,
			Apps: appsResource.Apps,
		}, nil
	}
}

var the_apps = map[string]AppVersionsResource{
	"test-app": AppVersionsResource{
		Name: "test-app",
		Apps: map[string]AppResource{
			"0.2.0": AppResource{"test-app-0-2-0", "test-app", "0.2.0"},
			"1.2.3": AppResource{"test-app-1-2-3", "test-app", "1.2.3"},
			"1.3.9": AppResource{"test-app-1-3-9", "test-app", "1.3.9"},
		},
	},
}

type AppResource struct {
	ID      string
	Name    string
	Version string
}

func (AppResource) HandleGET(parentIDs map[string]string, version string) (*AppResource, error) {
	println("PARENT IDS:")
	for k, v := range parentIDs {
		println("\t", k, "=", v)
	}
	name := parentIDs["apps"]
	if appResource, ok := the_apps[name]; !ok {
		return nil, Error404(name)
	} else if ver, ok := appResource.Apps[version]; !ok {
		return nil, Error404(name + " v" + version)
	} else {
		return &ver, nil
	}
}
