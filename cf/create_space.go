package cf

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cenkalti/backoff"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pivotal-cf/perm-test/cf/internal"
)

// CreateSpaceIfNotExists creates a space in CloudFoundry using the V2 API
// It uses an exponential backoff strategy, returning early if it successfully creates
// a space or the space already exists
func CreateSpaceIfNotExists(logger lager.Logger, cfClient *cfclient.Client, spaceName string, orgGUID string) (*cfclient.Space, error) {
	logger.Debug("creating-space")
	spaceRequest := cfclient.SpaceRequest{
		Name:             spaceName,
		OrganizationGuid: orgGUID,
	}
	var (
		err   error
		space cfclient.Space
	)

	operation := func() error {
		space, err = cfClient.CreateSpace(spaceRequest)
		switch e := err.(type) {
		case nil:
			return nil

		case cfclient.CloudFoundryErrors:
			if len(e.Errors) == 0 {
				return err
			}

			for _, cfError := range e.Errors {
				if cfError.ErrorCode == internal.SpaceNameTaken {
					return nil
				}
			}
		case cfclient.CloudFoundryError:
			if e.ErrorCode == internal.SpaceNameTaken {
				return nil
			}

			return err
		default:
			return err
		}

		return err
	}

	err = backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, step time.Duration) {
		logger.Error("failed-to-create-space", err, lager.Data{
			"backoff.step": step.String(),
		})
	})

	if err != nil {
		logger.Error("finally-failed-to-create-space", err)
		return nil, err
	}

	return &space, nil
}
