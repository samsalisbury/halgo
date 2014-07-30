package main

import (
	"github.com/samsalisbury/halgo"
)

type RootResource struct {
	Welcome string
	Version string
	Apps    *AppsResource
	Health  *HealthResource
}

func (RootResource) ChildResources() []halgo.Resource {
	return []halgo.Resource{AppsResource{}, HealthResource{}}
}

func (RootResource) GET() (*RootResource, error) {
	return &RootResource{
		Welcome: "Welcome to the deployment service API",
		Version: "0.0.110",
	}, nil
}

type HealthResource struct {
	Hello string
}

func (HealthResource) GET() (*HealthResource, error) {
	return &HealthResource{
		Hello: "Feelin' good!",
	}, nil
}

func (HealthResource) ChildResources() []halgo.Resource {
	return nil
}

type AppsResource struct {
	NumberOfApps int
	AppVersions  map[string]AppVersionsResource
}

func (AppsResource) GET() (*AppsResource, error) {
	return &AppsResource{
		NumberOfApps: len(the_apps),
		AppVersions:  the_apps,
	}, nil
}

func (AppsResource) ChildResources() []halgo.Resource {
	return []halgo.Resource{AppResource{}}
}

type App struct {
	Name string
	Apps map[string]AppVersion
}

func (App) GET(name string) (*App, error) {
	if appsResource, ok := the_apps[name]; !ok {
		return nil, halgo.Error404(name)
	} else {
		return &App{
			Name:     name,
			Versions: appsResource.Versions,
		}, nil
	}
}

var the_apps = map[string]AppVersionsResource{
	"test-app": AppVersionsResource{
		Name: "test-app",
		Versions: map[string]App{
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

func (AppResource) GET(parentIDs map[string]string, version string) (*AppResource, error) {
	return nil, nil
	// name := parentIDs["app"]
	// if appResource, ok := the_apps[name]; !ok {
	// 	return nil, Error404(name)
	// } else if ver, ok := appResource.Apps[version]; !ok {
	// 	return nil, Error404(name)
	// } else {
	// 	return &ver, nil
	// }
}

func (AppResource) ChildResources() []halgo.Resource {
	return nil
}
