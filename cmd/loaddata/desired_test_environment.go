package main

import (
	"context"
	"fmt"
	"sync"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pivotal-cf/perm-test/cf"
	"golang.org/x/sync/semaphore"
)

type DesiredTestEnvironment struct {
	UserGUID          string
	OrgCount          int
	SpacesPerOrgCount int
	AppsPerSpaceCount int
}

func (e *DesiredTestEnvironment) Create(ctx context.Context, logger lager.Logger, sem *semaphore.Weighted, cfClient *cfclient.Client) {
	user, err := cf.CreateUser(logger, cfClient, e.UserGUID)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < e.OrgCount; i++ {
		err = sem.Acquire(ctx, 1)
		if err != nil {
			logger.Error("failed-to-acquire-semaphore", err)
			panic(err)
		}

		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, logger lager.Logger, i int) {
			defer wg.Done()
			defer sem.Release(1)

			err = createAndPopulateOrgInTestEnvironment(logger, cfClient, i, user.Guid, e.SpacesPerOrgCount, e.AppsPerSpaceCount)
			if err != nil {
				panic(err)
			}
		}(ctx, &wg, logger, i)
	}
	wg.Wait()
}

func createAndPopulateOrgInTestEnvironment(logger lager.Logger, cfClient *cfclient.Client, i int, userGUID string, spacesPerOrgCount int, appsPerSpaceCount int) error {
	orgName := fmt.Sprintf("perm-test-org-%d", i)
	logger = logger.WithData(lager.Data{
		"org.name": orgName,
	})

	org, err := cf.CreateOrg(logger, cfClient, orgName)
	if err != nil {
		return err
	}

	logger = logger.WithData(lager.Data{
		"user.guid": userGUID,
	})

	err = cf.AssociateUserWithOrg(logger, cfClient, userGUID, org.Guid)
	if err != nil {
		return err
	}

	for j := 0; j < spacesPerOrgCount; j++ {
		spaceName := fmt.Sprintf("perm-test-space-%d-in-org-%d", j, i)
		logger = logger.WithData(lager.Data{
			"space.name": spaceName,
		})

		space, err := cf.CreateSpace(logger, cfClient, spaceName, org.Guid)
		if err != nil {
			return err
		}

		err = cf.MakeUserSpaceDeveloper(logger, cfClient, userGUID, space.Guid)
		if err != nil {
			return err
		}

		for k := 0; k < appsPerSpaceCount; k++ {
			appName := fmt.Sprintf("perm-test-app-%d-in-space-%d-in-org-%d", k, j, i)
			logger = logger.WithData(lager.Data{
				"app.name": appName,
			})

			err = cf.CreateApp(logger, cfClient, appName, space.Guid)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
