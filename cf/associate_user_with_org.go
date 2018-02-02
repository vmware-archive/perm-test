package cf

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cenkalti/backoff"
	"github.com/cloudfoundry-community/go-cfclient"
)

func AssociateUserWithOrg(logger lager.Logger, cfClient *cfclient.Client, userGUID string, orgGUID string) error {
	logger.Debug("associating-user-with-org")

	var err error
	operation := func() error {
		_, err = cfClient.AssociateOrgUser(orgGUID, userGUID)
		return err
	}
	err = backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, step time.Duration) {
		logger.Error("failed-to-associate-user-with-org", err, lager.Data{
			"backoff.step": step.String(),
		})
	})

	if err != nil {
		logger.Error("finally-failed-to-associate-user-with-org", err)
	}
	return err
}
