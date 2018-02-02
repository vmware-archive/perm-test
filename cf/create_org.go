package cf

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cenkalti/backoff"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pivotal-cf/perm-test/cf/internal"
)

// CreateOrgIfNotExists creates an org in CloudFoundry using the V2 API
// It uses an exponential backoff strategy, returning early if it successfully creates
// an org or the org already exists
func CreateOrgIfNotExists(logger lager.Logger, cfClient *cfclient.Client, orgName string) (*cfclient.Org, error) {
	logger.Debug("creating-org", lager.Data{
		"name": orgName,
	})

	orgRequest := cfclient.OrgRequest{
		Name: orgName,
	}

	var (
		org cfclient.Org
		err error
	)
	operation := func() error {
		org, err = cfClient.CreateOrg(orgRequest)
		switch e := err.(type) {
		case nil:
			return nil

		case cfclient.CloudFoundryErrors:
			if len(e.Errors) == 0 {
				return err
			}

			cfError := e.Errors[0]
			if cfError.ErrorCode == internal.OrganizationNameTaken {
				return nil
			}
		case cfclient.CloudFoundryError:
			if e.ErrorCode == internal.OrganizationNameTaken {
				return nil
			}

			return err
		default:
			return err
		}

		return err
	}

	err = backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, step time.Duration) {
		logger.Error("failed-to-create-org", err, lager.Data{
			"backoff.step": step.String(),
		})
	})
	if err != nil {
		logger.Error("finally-failed-to-create-org", err)
		return nil, err
	}
	return &org, nil
}
