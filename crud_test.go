package halgo

func (entity *RootResource) Manifest() error {
	(*entity) = RootResource{
		Welcome: "Welcome to the deployment service API",
		Version: "0.0.110",
		Apps:    nil,
	}
	return nil
}

func (entity *Health) Manifest() error {
	(*entity) = Health{
		Hello: "Feelin' good!",
	}
	return nil
}

func (entity *Apps) Manifest() error {
	(*entity) = Apps{
		NumberOfApps: len(the_apps),
		Apps:         the_apps,
	}
	return nil
}

func (entity *Apps) Collection() (AppsList, error) {
	return nil, nil
}

func (entity *App) Manifest(parent *Apps, id string) error {
	if app, ok := parent.Apps[id]; ok {
		(*entity) = app
	}
	return nil
}

func (entity *App) Collection(parent *Apps, id string) (map[string]AppVersion, error) {
	return the_apps[id].Versions, nil
}

// Manifest should try to find and load the entity. If any parents are missing,
// it should return a 404 on that parent. If the parents are there, but this
// item is not, it should leave a as nil. Otherwise it should populate a with
// the stored entity.
func (a *AppVersion) Manifest(parent *App, id string) error {
	if ver, ok := parent.Versions[id]; ok {
		(*a) = ver
	}
	return nil
}

// Write could throw either a 404 if a parent is missing (if that is important)
// or a 500 if the write fails. Or a 409 if it's read-only etc.
func (a *AppVersion) Write(parent *App, id string) error {
	parent.Versions[id] = *a
	return nil
}

func (a *AppVersion) Validate(parent *App, id string) error {
	// Anything this returns will be interpreted as a validation error (e.g. 400/409/4xx?)
	return nil
}

func (a *AppVersion) Process(parent *App, payload interface{}) (interface{}, error) {
	// This would be the POST handler
	return nil, Error("Process not implemented.")
}

func (a *AppVersion) Delete(parent *App, id string) error {
	delete(parent.Versions, id)
	return nil
}
