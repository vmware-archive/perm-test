package main

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pivotal-cf/perm-test/cf"
	"github.com/pivotal-cf/perm-test/cmd"
	"github.com/satori/go.uuid"
	"golang.org/x/sync/semaphore"
)

type DesiredExternalEnvironment struct {
	SpacesPerOrgCount      int
	AppsPerSpaceCount      int
	OrgCount               int
	UserCount              int
	UserOrgDistributions   []cmd.UserOrgDistribution
	UserSpaceDistributions []cmd.UserSpaceDistribution
}

func (e *DesiredExternalEnvironment) Create(ctx context.Context, logger lager.Logger, sem *semaphore.Weighted, cfClient *cfclient.Client) {
	// Create a bunch of orgs/spaces/apps
	//orgsBufferSize := e.OrgCount * 2
	//orgsCreatedChan := make(chan *cfclient.Org, orgsBufferSize)
	//
	//spacesBufferSize := orgsBufferSize * e.SpacesPerOrgCount
	//spacesCreatedChan := make(chan *cfclient.Space, spacesBufferSize)

	//logger.Debug("creating-orgs-spaces-and-apps", lager.Data{
	//	"spaces-per-org-count": e.SpacesPerOrgCount,
	//	"apps-per-space-count": e.AppsPerSpaceCount,
	//	"org-count":            e.OrgCount,
	//})
	var err error
	var wg sync.WaitGroup

	//for i := 0; i < e.OrgCount; i++ {
	//	err = sem.Acquire(ctx, 1)
	//	if err != nil {
	//		logger.Error("failed-to-acquire-semaphore", err)
	//		panic(err)
	//	}
	//
	//	wg.Add(1)
	//	go func(ctx context.Context, wg *sync.WaitGroup, sem *semaphore.Weighted, logger lager.Logger, i int) {
	//		defer wg.Done()
	//		defer sem.Release(1)
	//
	//		orgName := fmt.Sprintf("perm-external-org-%d", i)
	//		logger = logger.WithData(lager.Data{
	//			"org.name": orgName,
	//		})
	//		org, err := cf.CreateOrgIfNotExists(logger, cfClient, orgName)
	//		if err != nil {
	//			panic(err)
	//		}
	//
	//		orgsCreatedChan <- org
	//
	//		for j := 0; j < e.SpacesPerOrgCount; j++ {
	//			spaceName := fmt.Sprintf("perm-external-space-%d-in-org-%d", j, i)
	//			logger = logger.WithData(lager.Data{
	//				"space.name": spaceName,
	//			})
	//
	//			space, err := cf.CreateSpaceIfNotExists(logger, cfClient, spaceName, org.Guid)
	//			if err != nil {
	//				panic(err)
	//			}
	//
	//			spacesCreatedChan <- space
	//
	//			for k := 0; k < e.AppsPerSpaceCount; k++ {
	//				appName := fmt.Sprintf("perm-external-app-%d-in-space-%d-in-org-%d", k, j, i)
	//				logger = logger.WithData(lager.Data{
	//					"app.name": appName,
	//				})
	//
	//				err = cf.CreateAppIfNotExists(logger, cfClient, appName, space.Guid)
	//				if err != nil {
	//					panic(err)
	//				}
	//			}
	//		}
	//
	//	}(ctx, &wg, sem, logger, i)
	//}
	//wg.Wait()
	//close(orgsCreatedChan)
	//close(spacesCreatedChan)

	//var orgsCreated []*cfclient.Org
	//for org := range orgsCreatedChan {
	//	orgsCreated = append(orgsCreated, org)
	//}

	logger.Info("listing-orgs")
	orgsCreated, err := cfClient.ListOrgs()
	if err != nil {
		logger.Error("failed-to-list-orgs", err)
		panic(err)
	}
	logger.Info("done-listing-orgs")

	//var spacesCreated []*cfclient.Space
	//for space := range spacesCreatedChan {
	//	spacesCreated = append(spacesCreated, space)
	//}

	logger.Info("listing-spaces")
	spacesCreated, err := cfClient.ListSpaces()
	if err != nil {
		logger.Error("failed-to-list-spaces", err)
		panic(err)
	}
	logger.Info("done-listing-spaces")

	logger.Debug("creating-users-and-assigning-roles", lager.Data{
		"spaces-per-org-count":    e.SpacesPerOrgCount,
		"apps-per-space-count":    e.AppsPerSpaceCount,
		"org-count":               e.OrgCount,
		"user-count":              e.UserCount,
		"user-org-distribution":   e.UserOrgDistributions,
		"user-space-distribution": e.UserSpaceDistributions,
	})
	// Create a bunch of users
	// For every user
	//  Calculate the number of orgs it should see
	//    Randomly assign an org role for that many orgs
	//  Calculate the number of spaces it should see
	//    Randomly assign a space role for that many spaces
	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	for i := 0; i < e.UserCount; i++ {
		err = sem.Acquire(ctx, 1)
		if err != nil {
			logger.Error("failed-to-acquire-semaphore", err)
			panic(err)
		}

		numOrgAssignments := cmd.ChooseNumOrgAssignments(r, e.UserOrgDistributions)
		orgs := cmd.RandomlyChooseOrgs(r, orgsCreated, numOrgAssignments)

		numSpaceAssignments := cmd.ChooseNumSpaceAssignments(r, e.UserSpaceDistributions)
		spaces := cmd.RandomlyChooseSpaces(r, spacesCreated, numSpaceAssignments)

		logger.Debug("creating-user-and-assigning-roles", lager.Data{
			"i": i,
			"numSpaceAssignments": numSpaceAssignments,
			"numSpaces":           len(spaces),
			"numOrgAssignments":   numOrgAssignments,
			"numOrgs":             len(orgs),
		})
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, sem *semaphore.Weighted, logger lager.Logger, i int, r *rand.Rand, orgs []cfclient.Org, spaces []cfclient.Space) {
			defer wg.Done()
			defer sem.Release(1)

			userUUID := uuid.NewV4()

			logger = logger.WithData(lager.Data{
				"user.guid": userUUID.String(),
			})
			user, err := cf.CreateUser(logger, cfClient, userUUID.String())
			if err != nil {
				panic(err)
			}

			logger.Debug("assigning-space-roles", lager.Data{
				"space.count": len(spaces),
			})
			for _, space := range spaces {
				spaceLogger := logger.WithData(lager.Data{
					"org.guid":   space.OrganizationGuid,
					"space.name": space.Name,
				})

				spaceLogger.Debug("associating-user-with-org-for-space")
				err = cf.AssociateUserWithOrg(logger, cfClient, user.Guid, space.OrganizationGuid)
				if err != nil {
					panic(err)
				}

				spaceLogger.Debug("making-user-space-developer")
				err = cf.MakeUserSpaceDeveloper(logger, cfClient, user.Guid, space.Guid)
				if err != nil {
					panic(err)
				}
			}

			logger.Debug("assigning-org-roles", lager.Data{
				"org.count": len(orgs),
			})
			for _, org := range orgs {
				orgLogger := logger.WithData(lager.Data{
					"org.name": org.Name,
				})
				orgLogger.Debug("associating-user-with-org")

				err = cf.AssociateUserWithOrg(logger, cfClient, user.Guid, org.Guid)
				if err != nil {
					panic(err)
				}
			}
		}(ctx, &wg, sem, logger, i, r, orgs, spaces)
	}
}
