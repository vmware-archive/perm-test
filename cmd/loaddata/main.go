package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pivotal-cf/perm-test/cmd"
	"gopkg.in/yaml.v2"

	"net/http"
	"sync"

	"golang.org/x/sync/semaphore"
)

const (
	NumParallelWorkers     = 12
	CloudControllerTimeout = 3 * time.Second
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: loaddata <path/to/config.yml>")
		os.Exit(2)
	}

	configPath := os.Args[1]
	contents, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config file: %s\n", err.Error())
		panic(err)
	}

	var config cmd.LoadDataConfig
	err = yaml.Unmarshal(contents, &config)
	if err != nil {
		fmt.Printf("Failed to parse config file data: %s\n", err.Error())
		panic(err)
	}

	logger := config.NewLogger("perm-loaddata")
	logger.Info("starting")

	httpClient := &http.Client{
		Timeout: CloudControllerTimeout,
	}

	cfClientConfig := &cfclient.Config{
		ApiAddress:        config.CloudControllerConfig.URL,
		Username:          config.CloudControllerConfig.ClientID,
		Password:          config.CloudControllerConfig.ClientSecret,
		SkipSslValidation: true,
		HttpClient:        httpClient,
	}
	cfClient, err := cfclient.NewClient(cfClientConfig)
	if err != nil {
		logger.Error("failed-to-make-cf-client", err)
		panic(err)
	}

	defer logger.Info("finished")

	ctx := context.Background()
	sem := semaphore.NewWeighted(NumParallelWorkers)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()

		e := &DesiredTestEnvironment{
			UserGUID:          config.TestDataConfig.TestEnvironmentConfig.UserGUID,
			OrgCount:          config.TestDataConfig.TestEnvironmentConfig.OrgCount,
			SpacesPerOrgCount: config.TestDataConfig.SpacesPerOrgCount,
			AppsPerSpaceCount: config.TestDataConfig.AppsPerSpaceCount,
		}

		e.Create(ctx, logger.Session("create-test-environment"), sem, cfClient)
	}()

	go func() {
		defer wg.Done()

		e := &DesiredExternalEnvironment{
			UserCount:         config.TestDataConfig.ExternalEnvironmentConfig.UserCount,
			OrgCount:          config.TestDataConfig.ExternalEnvironmentConfig.OrgCount,
			SpacesPerOrgCount: config.TestDataConfig.SpacesPerOrgCount,
			AppsPerSpaceCount: config.TestDataConfig.AppsPerSpaceCount,
			Lambda:            20,
		}

		e.Create(ctx, logger.Session("create-external-environment"), sem, cfClient)
	}()

	wg.Wait()

}
