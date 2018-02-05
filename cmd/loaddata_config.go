package cmd

import (
	"errors"
	"os"

	"code.cloudfoundry.org/lager"
)

type LoadDataConfig struct {
	LogLevel              string                `yaml:"log_level"`
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
	AppsPerSpaceCount int `yaml:"apps_per_space_count"`
	SpacesPerOrgCount int `yaml:"spaces_per_org_count"`

	TestEnvironmentConfig     TestEnvironmentConfig     `yaml:"test_environment"`
	ExternalEnvironmentConfig ExternalEnvironmentConfig `yaml:"external_environment"`
}

type TestEnvironmentConfig struct {
	UserGUID string `yaml:"user_guid"`
	OrgCount int    `yaml:"org_count"`
}

type ExternalEnvironmentConfig struct {
	OrgCount               int                     `yaml:"org_count"`
	UserCount              int                     `yaml:"user_count"`
	UserOrgDistributions   []UserOrgDistribution   `yaml:"user_org_distribution"`
	UserSpaceDistributions []UserSpaceDistribution `yaml:"user_space_distribution"`
}

type UserOrgDistribution struct {
	PercentUsers float64 `yaml:"percent_users"`
	NumOrgs      int     `yaml:"num_orgs"`
}

type UserSpaceDistribution struct {
	PercentUsers float64 `yaml:"percent_users"`
	NumSpaces    int     `yaml:"num_spaces"`
}

func (c *LoadDataConfig) NewLogger(component string) lager.Logger {
	var l lager.LogLevel

	switch c.LogLevel {
	case "debug":
		l = lager.DEBUG
	case "info":
		l = lager.INFO
	case "error":
		l = lager.ERROR
	case "fatal":
		l = lager.FATAL
	default:
		l = lager.INFO
	}

	sink := lager.NewWriterSink(os.Stdout, l)
	logger := lager.NewLogger(component)
	logger.RegisterSink(sink)

	return logger
}

func (c *LoadDataConfig) Validate() error {
	var p float64
	for _, d := range c.TestDataConfig.ExternalEnvironmentConfig.UserOrgDistributions {
		p += d.PercentUsers

		if d.NumOrgs > c.TestDataConfig.TestEnvironmentConfig.OrgCount {
			return errors.New("error in user_org_distribution: users in external environment should not have access to more orgs than test user")
		}
	}
	if p != 1 {
		return errors.New("error in user_org_distribution: percentage of users must sum to 1")
	}

	p = 0.0
	for _, d := range c.TestDataConfig.ExternalEnvironmentConfig.UserSpaceDistributions {
		p += d.PercentUsers

		if d.NumSpaces > (c.TestDataConfig.TestEnvironmentConfig.OrgCount * c.TestDataConfig.SpacesPerOrgCount) {
			return errors.New("error in user_space_distribution: users in external environment should not have access to more spaces than test user")
		}
	}
	if p != 1 {
		return errors.New("error in user_space_distribution: percentage of users must sum to 1")
	}

	return nil
}
