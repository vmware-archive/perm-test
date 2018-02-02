package cf

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cenkalti/backoff"
	"github.com/cloudfoundry-community/go-cfclient"
)

func MakeUserSpaceDeveloper(logger lager.Logger, cfClient *cfclient.Client, userGUID string, spaceGUID string) error {
	logger.Debug("making-user-space-developer")
	r := cfClient.NewRequest("PUT", fmt.Sprintf("/v2/spaces/%s/developers/%s", spaceGUID, userGUID))
	operation := func() error {
		resp, err := cfClient.DoRequest(r)

		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusCreated {
			err = fmt.Errorf("Incorrect status code (%d)", resp.StatusCode)
			return err
		}

		return nil
	}
	err := backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, step time.Duration) {
		logger.Error("failed-to-make-user-space-developer", err, lager.Data{
			"backoff.step": step.String(),
		})
	})

	if err != nil {
		logger.Error("finally-failed-to-make-user-space-developer", err)
	}
	return err
}
