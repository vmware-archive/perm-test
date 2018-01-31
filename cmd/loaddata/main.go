package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/perm-test/cmd"
	"gopkg.in/yaml.v2"

	"bytes"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/satori/go.uuid"

	"golang.org/x/sync/semaphore"
)

const NumParallelWorkers = 12

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

	cfClientConfig := &cfclient.Config{
		ApiAddress:        config.CloudControllerConfig.URL,
		Username:          config.CloudControllerConfig.ClientID,
		Password:          config.CloudControllerConfig.ClientSecret,
		SkipSslValidation: true,
	}
	cfClient, err := cfclient.NewClient(cfClientConfig)
	if err != nil {
		logger.Fatal("failed-to-make-cf-client", err)
	}

	userID := uuid.NewV4()
	userRequest := cfclient.UserRequest{
		Guid: userID.String(),
	}
	logger.Debug("creating-user", lager.Data{
		"guid": userID.String(),
	})
	user, err := cfClient.CreateUser(userRequest)
	if err != nil {
		logger.Fatal("failed-to-create-cf-user", err)
	}

	orgName := "test-org"
	orgRequest := cfclient.OrgRequest{
		Name: orgName,
	}
	logger.Debug("creating-org", lager.Data{
		"name": orgName,
	})
	org, err := cfClient.CreateOrg(orgRequest)
	if err != nil {
		logger.Fatal("failed-to-create-cf-org", err)
	}

	logger = logger.WithData(lager.Data{
		"user.guid": userID.String(),
		"org.name":  orgName,
	})
	logger.Debug("associating-user-with-org")
	_, err = cfClient.AssociateOrgUser(org.Guid, userID.String())
	if err != nil {
		fmt.Printf("Failed to make user of an org: %s\n", err.Error())
	}

	wg := sync.WaitGroup{}

	sem := semaphore.NewWeighted(NumParallelWorkers)
	ctx := context.Background()

	defer logger.Debug("finished")
	for i := 0; i < config.TestDataConfig.SpaceCount; i++ {
		err = sem.Acquire(ctx, 1)
		if err != nil {
			logger.Fatal("failed-to-acquire-semaphore", err)
		}

		wg.Add(1)
		go func(wg *sync.WaitGroup, logger lager.Logger, org cfclient.Org, i int) {
			defer wg.Done()
			defer sem.Release(1)

			spaceName := fmt.Sprintf("perm-test-space-%d", i)
			logger = logger.WithData(lager.Data{
				"space.name": spaceName,
			})
			logger.Debug("creating-space")
			spaceRequest := cfclient.SpaceRequest{
				Name:             spaceName,
				OrganizationGuid: org.Guid,
			}
			space, err := cfClient.CreateSpace(spaceRequest)
			if err != nil {
				logger.Fatal("failed-to-create-space", err)
			}

			logger.Debug("making-user-space-developer")
			r := cfClient.NewRequest("PUT", fmt.Sprintf("/v2/spaces/%s/developers/%s", space.Guid, user.Guid))
			resp, err := cfClient.DoRequest(r)
			if err != nil {
				logger.Fatal("failed-to-make-user-space-developer", err)
			}

			if resp.StatusCode != http.StatusCreated {
				err = fmt.Errorf("Incorrect status code (%d)", resp.StatusCode)
				logger.Fatal("failed-to-make-user-space-developer", err)
			}

			for j := 0; j < config.TestDataConfig.AppsPerSpaceCount; j++ {
				buf := bytes.NewBuffer(nil)
				appName := fmt.Sprintf("perm-test-app-%d-in-space-%d", j, i)

				logger = logger.WithData(lager.Data{
					"app.name": appName,
				})
				req := NewCreateAppRequest(appName, SpaceGUID(space.Guid))

				err := json.NewEncoder(buf).Encode(req)
				if err != nil {
					logger.Fatal("failed-to-create-app-request", err)
				}

				logger.Debug("creating-app")
				r := cfClient.NewRequestWithBody("POST", "/v3/apps", buf)
				resp, err := cfClient.DoRequest(r)
				if err != nil {
					logger.Fatal("failed-to-create-app", err)
				}

				if resp.StatusCode != http.StatusCreated {
					err = fmt.Errorf("Incorrect status code (%d)", resp.StatusCode)
					logger.Fatal("failed-to-create-app", err)
				}
			}
		}(&wg, logger, org, i)
	}

	wg.Wait()
}

type SpaceGUID string

func NewCreateAppRequest(appName string, spaceGUID SpaceGUID) *CreateAppRequestBody {
	return &CreateAppRequestBody{
		Name: appName,
		Relationships: SpaceRelationship{
			Space: Space{
				Data: Data{GUID: string(spaceGUID)},
			},
		},
	}
}

type CreateAppRequestBody struct {
	Name          string            `json:"name"`
	Relationships SpaceRelationship `json:"relationships"`
}

type SpaceRelationship struct {
	Space Space `json:"space"`
}

type Space struct {
	Data Data `json:"data"`
}

type Data struct {
	GUID string `json:"guid"`
}
