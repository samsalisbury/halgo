package test

type Apps struct {
	NumberOfApps int
	Apps         []App
}

func (Apps) GET() (*Apps, error) {
	return &Apps{
		NumberOfApps: len(the_apps),
		Apps:         the_apps,
	}, nil
}
