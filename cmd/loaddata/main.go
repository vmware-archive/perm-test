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

	"github.com/cenkalti/backoff"

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
		logger.Fatal("failed-to-make-cf-client", err)
	}

	userID := config.TestDataConfig.SystemUnderTestConfig.UserGUID
	userRequest := cfclient.UserRequest{
		Guid: userID,
	}
	logger.Debug("creating-user", lager.Data{
		"guid": userID,
	})
	user, err := cfClient.CreateUser(userRequest)
	if err != nil {
		logger.Fatal("failed-to-create-cf-user", err)
		os.Exit(1)
	}

	wg := sync.WaitGroup{}

	sem := semaphore.NewWeighted(NumParallelWorkers)
	ctx := context.Background()

	defer logger.Debug("finished")
	for i := 0; i < config.TestDataConfig.SystemUnderTestConfig.OrgCount; i++ {
		err = sem.Acquire(ctx, 1)
		if err != nil {
			logger.Fatal("failed-to-acquire-semaphore", err)
		}

		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, logger lager.Logger, i int) {
			defer wg.Done()
			defer sem.Release(1)

			orgName := fmt.Sprintf("perm-test-org-%d", i)
			orgRequest := cfclient.OrgRequest{
				Name: orgName,
			}
			logger.Debug("creating-org", lager.Data{
				"name": orgName,
			})
			var org cfclient.Org
			operation := func() error {
				org, err = cfClient.CreateOrg(orgRequest)
				if err != nil {
					return err
				}
				return nil
			}

			err = backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, step time.Duration) {
				logger.Error("failed-to-create-org", err, lager.Data{
					"backoff.step": step.String(),
				})
			})
			if err != nil {
				logger.Fatal("finally-failed-to-create-org", err)
				return
			}
			logger = logger.WithData(lager.Data{
				"user.guid": userID,
				"org.name":  orgName,
			})
			logger.Debug("associating-user-with-org")
			operation = func() error {
				_, err = cfClient.AssociateOrgUser(org.Guid, userID)
				return err
			}
			err = backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, step time.Duration) {
				logger.Error("failed-to-associate-user-with-org", err, lager.Data{
					"backoff.step": step.String(),
				})
			})
			if err != nil {
				logger.Fatal("finally-failed-to-associate-user-with-org", err)
				return
			}

			for j := 0; j < config.TestDataConfig.SpacesPerOrgCount; j++ {
				spaceName := fmt.Sprintf("perm-test-space-%d-in-org-%d", j, i)
				logger = logger.WithData(lager.Data{
					"space.name": spaceName,
				})

				space, err := cmd.CreateSpace(logger, cfClient, spaceName, org.Guid)
				if err != nil {
					panic(err)
				}

				err = cmd.MakeUserSpaceDeveloper(logger, cfClient, user.Guid, space.Guid)
				if err != nil {
					panic(err)
				}

				for k := 0; k < config.TestDataConfig.AppsPerSpaceCount; k++ {
					appName := fmt.Sprintf("perm-test-app-%d-in-space-%d-in-org-%d", k, j, i)
					logger = logger.WithData(lager.Data{
						"app.name": appName,
					})

					err = cmd.CreateApp(logger, cfClient, appName, space.Guid)
					if err != nil {
						panic(err)
					}
				}
			}
		}(ctx, &wg, logger, i)
	}

	wg.Wait()
}
