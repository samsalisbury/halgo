package test

type RootResource struct {
	Welcome string
	Version string
	Apps    *Apps
	Health  *HealthResource
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
