package test

type Apps struct {
	NumberOfApps int
	Apps         []App
}

func (Apps) HandleGET() (*Apps, error) {
	return &Apps{
		NumberOfApps: len(the_apps),
		Apps:         the_apps,
	}, nil
}
