package test

import (
	"github.com/samsalisbury/halgo"
)

func (RootResource) ChildResources() []halgo.Resource {
	return []Resource{
		AppsResource{},
		HealthResource{},
	}
}

func (HealthResource) ChildResources() []halgo.Resource {
	return nil
}

func (Apps) ChildResources() []halgo.Resource {
	return []Resource{
		App{},
	}
}

func (App) ChildResources() []halgo.Resource {
	return []Resource{
		AppVersion{},
	}
}

func (AppVersion) ChildResources() []halgo.Resource {
	return nil
}
