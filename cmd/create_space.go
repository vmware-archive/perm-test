package cmd

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cenkalti/backoff"
	"github.com/cloudfoundry-community/go-cfclient"
)

func CreateSpace(logger lager.Logger, cfClient *cfclient.Client, spaceName string, orgGUID string) (*cfclient.Space, error) {
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
		if err != nil {
			return err
		}
		return nil
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
