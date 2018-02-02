package cf

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cenkalti/backoff"
	"github.com/cloudfoundry-community/go-cfclient"
)

func CreateOrg(logger lager.Logger, cfClient *cfclient.Client, orgName string) (*cfclient.Org, error) {
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
		logger.Error("finally-failed-to-create-org", err)
		return nil, err
	}
	return &org, nil
}
