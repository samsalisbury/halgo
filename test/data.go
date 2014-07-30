package test

var the_apps = map[string]AppVersionsResource{
	"test-app": App{
		Name: "test-app",
		Versions: map[string]AppVersion{
			"0.2.0": AppResource{"test-app-0-2-0", "test-app", "0.2.0"},
			"1.2.3": AppResource{"test-app-1-2-3", "test-app", "1.2.3"},
			"1.3.9": AppResource{"test-app-1-3-9", "test-app", "1.3.9"},
		},
	},
}
