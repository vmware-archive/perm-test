package main

import (
	"context"
	"fmt"
	"math"
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
	orgsBufferSize := e.OrgCount * 2
	orgsCreatedChan := make(chan *cfclient.Org, orgsBufferSize)

	spacesBufferSize := orgsBufferSize * e.SpacesPerOrgCount
	spacesCreatedChan := make(chan *cfclient.Space, spacesBufferSize)

	logger.Debug("creating-orgs-spaces-and-apps", lager.Data{
		"spaces-per-org-count": e.SpacesPerOrgCount,
		"apps-per-space-count": e.AppsPerSpaceCount,
		"org-count":            e.OrgCount,
	})
	var err error
	var wg sync.WaitGroup
	for i := 0; i < e.OrgCount; i++ {
		err = sem.Acquire(ctx, 1)
		if err != nil {
			logger.Error("failed-to-acquire-semaphore", err)
			panic(err)
		}

		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, sem *semaphore.Weighted, logger lager.Logger, i int) {
			defer wg.Done()
			defer sem.Release(1)

			orgName := fmt.Sprintf("perm-external-org-%d", i)
			logger = logger.WithData(lager.Data{
				"org.name": orgName,
			})
			org, err := cf.CreateOrgIfNotExists(logger, cfClient, orgName)
			if err != nil {
				panic(err)
			}

			orgsCreatedChan <- org

			for j := 0; j < e.SpacesPerOrgCount; j++ {
				spaceName := fmt.Sprintf("perm-external-space-%d-in-org-%d", j, i)
				logger = logger.WithData(lager.Data{
					"space.name": spaceName,
				})

				space, err := cf.CreateSpaceIfNotExists(logger, cfClient, spaceName, org.Guid)
				if err != nil {
					panic(err)
				}

				spacesCreatedChan <- space

				for k := 0; k < e.AppsPerSpaceCount; k++ {
					appName := fmt.Sprintf("perm-external-app-%d-in-space-%d-in-org-%d", k, j, i)
					logger = logger.WithData(lager.Data{
						"app.name": appName,
					})

					err = cf.CreateAppIfNotExists(logger, cfClient, appName, space.Guid)
					if err != nil {
						panic(err)
					}
				}
			}

		}(ctx, &wg, sem, logger, i)
	}
	wg.Wait()
	close(orgsCreatedChan)
	close(spacesCreatedChan)

	var orgsCreated []*cfclient.Org
	for org := range orgsCreatedChan {
		orgsCreated = append(orgsCreated, org)
	}

	var spacesCreated []*cfclient.Space
	for space := range spacesCreatedChan {
		spacesCreated = append(spacesCreated, space)
	}

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

		numOrgAssignments := chooseNumOrgAssignments(r, e.UserOrgDistributions)
		orgs := randomlyChooseOrgs(r, orgsCreated, numOrgAssignments)

		numSpaceAssignments := chooseNumSpaceAssignments(r, e.UserSpaceDistributions)
		spaces := randomlyChooseSpaces(r, spacesCreated, numSpaceAssignments)

		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, sem *semaphore.Weighted, logger lager.Logger, i int, r *rand.Rand, orgs []*cfclient.Org, spaces []*cfclient.Space) {
			defer wg.Done()
			defer sem.Release(1)

			userUUID := uuid.NewV4()

			user, err := cf.CreateUser(logger, cfClient, userUUID.String())
			if err != nil {
				panic(err)
			}

			logger = logger.WithData(lager.Data{
				"user.guid": user.Guid,
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

// chooseNumOrgAssignments returns a number of organization assignments sampled
// from the distribution.
//
// It does this by figuring out which "bucket" a randomly sampled user belongs to
// and returning the number of orgs assigned to the bucket.
func chooseNumOrgAssignments(r *rand.Rand, distributions []cmd.UserOrgDistribution) uint {
	x := r.Float64()

	var cum float64
	for _, d := range distributions {
		if x > cum && x <= cum+d.PercentUsers {
			return uint(d.NumOrgs)
		}

		cum += d.PercentUsers
	}

	return 0
}

// chooseNumSpaceAssignments returns a number of space assignments sampled
// from the distribution.
//
// It does this by figuring out which "bucket" a randomly sampled user belongs to
// and returning the number of spaces assigned to the bucket.
func chooseNumSpaceAssignments(r *rand.Rand, distributions []cmd.UserSpaceDistribution) uint {
	x := r.Float64()

	var cum float64
	for _, d := range distributions {
		if x > cum && x <= cum+d.PercentUsers {
			return uint(d.NumSpaces)
		}

		cum += d.PercentUsers
	}

	return 0
}

// randomlyChooseOrgs returns a contiguous window of size num of orgs out of the slice
//
// It does this by randomly choosing an index between 0 and (len orgs - window size)
func randomlyChooseOrgs(r *rand.Rand, orgs []*cfclient.Org, num uint) []*cfclient.Org {
	maxIndex := int(math.Min(float64(len(orgs)-int(num)), float64(len(orgs))))

	idx := r.Intn(maxIndex)

	return orgs[idx:(idx + int(num))]
}

// randomlyChooseSpaces returns a contiguous window of size num of spaces out of the slice
//
// It does this by randomly choosing an index between 0 and (len spaces - window size)
func randomlyChooseSpaces(r *rand.Rand, spaces []*cfclient.Space, num uint) []*cfclient.Space {
	maxIndex := int(math.Min(float64(len(spaces)-int(num)), float64(len(spaces))))

	idx := r.Intn(maxIndex)

	return spaces[idx:(idx + int(num))]
}
