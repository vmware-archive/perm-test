package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
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
	logger.Debug("starting")

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

	userID := config.TestDataConfig.SystemUnderTestConfig.UserGUID
	user, err := cmd.CreateUser(logger, cfClient, userID)
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}

	sem := semaphore.NewWeighted(NumParallelWorkers)
	ctx := context.Background()

	defer logger.Debug("finished")
	for i := 0; i < config.TestDataConfig.SystemUnderTestConfig.OrgCount; i++ {
		err = sem.Acquire(ctx, 1)
		if err != nil {
			logger.Error("failed-to-acquire-semaphore", err)
			panic(err)
		}

		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, logger lager.Logger, i int) {
			defer wg.Done()
			defer sem.Release(1)

			err = createAndPopulateOrg(logger, cfClient, i, user.Guid, config.TestDataConfig.SpacesPerOrgCount, config.TestDataConfig.AppsPerSpaceCount)
			if err != nil {
				panic(err)
			}
		}(ctx, &wg, logger, i)
	}

	wg.Wait()
}

func createAndPopulateOrg(logger lager.Logger, cfClient *cfclient.Client, i int, userGUID string, spacesPerOrgCount int, appsPerSpaceCount int) error {
	orgName := fmt.Sprintf("perm-test-org-%d", i)
	org, err := cmd.CreateOrg(logger, cfClient, orgName)
	if err != nil {
		return err
	}

	logger = logger.WithData(lager.Data{
		"user.guid": userGUID,
		"org.name":  orgName,
	})

	err = cmd.AssociateUserWithOrg(logger, cfClient, userGUID, org.Guid)
	if err != nil {
		return err
	}

	for j := 0; j < spacesPerOrgCount; j++ {
		spaceName := fmt.Sprintf("perm-test-space-%d-in-org-%d", j, i)
		logger = logger.WithData(lager.Data{
			"space.name": spaceName,
		})

		space, err := cmd.CreateSpace(logger, cfClient, spaceName, org.Guid)
		if err != nil {
			return err
		}

		err = cmd.MakeUserSpaceDeveloper(logger, cfClient, userGUID, space.Guid)
		if err != nil {
			return err
		}

		for k := 0; k < appsPerSpaceCount; k++ {
			appName := fmt.Sprintf("perm-test-app-%d-in-space-%d-in-org-%d", k, j, i)
			logger = logger.WithData(lager.Data{
				"app.name": appName,
			})

			err = cmd.CreateApp(logger, cfClient, appName, space.Guid)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
