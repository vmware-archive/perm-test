package cmd

type LoadDataConfig struct {
	CloudControllerConfig CloudControllerConfig `yaml:"cloud_controller"`
	TestDataConfig        TestDataConfig        `yaml:"test_data"`
}

type CloudControllerConfig struct {
	URL          string `yaml:"url"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	CACert       string `yaml:"ca_cert"`
}

type TestDataConfig struct {
	SpaceCount        int `yaml:"space_count"`
	AppsPerSpaceCount int `yaml:"apps_per_space_count"`
}
