package test

type RootResource struct {
	Welcome string
	Version string
	Apps    *Apps
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
