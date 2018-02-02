package cf

import (
	"encoding/json"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cenkalti/backoff"
	"github.com/cloudfoundry-community/go-cfclient"
)

func OrgCount(logger lager.Logger, cfClient *cfclient.Client) (int, error) {
	return getCount(logger, cfClient, "/v3/organizations")
}

func SpaceCount(logger lager.Logger, cfClient *cfclient.Client) (int, error) {
	return getCount(logger, cfClient, "/v3/spaces")
}

func UserCount(logger lager.Logger, cfClient *cfclient.Client) (int, error) {
	req := cfClient.NewRequest("GET", "/v2/users")

	var (
		count int
		err   error
	)
	operation := func() error {
		resp, err := cfClient.DoRequest(req)
		if err != nil {
			logger.Error("failed-to-list-users", err)
			return err
		}

		var r v2CountResponse
		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&r)
		if err != nil {
			logger.Error("failed-to-parse-response-body", err)
			return err
		}

		count = r.TotalResults
		return nil
	}

	err = backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, step time.Duration) {
		logger.Error("failed-to-get-user-count", err, lager.Data{
			"backoff.step": step.String(),
		})
	})
	if err != nil {
		logger.Error("finally-failed-to-get-user-count", err)
		return 0, err
	}

	return count, err
}

type v2CountResponse struct {
	TotalResults int `json:"total_results"`
}

type v3CountResponse struct {
	Pagination v2CountResponse `json:"pagination"`
}

func getCount(logger lager.Logger, cfClient *cfclient.Client, path string) (int, error) {
	req := cfClient.NewRequest("GET", path)

	var (
		count int
		err   error
	)
	operation := func() error {
		resp, err := cfClient.DoRequest(req)
		if err != nil {
			logger.Error("failed-to-list", err)
			return err
		}

		var r v3CountResponse
		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&r)
		if err != nil {
			logger.Error("failed-to-parse-response-body", err)
			return err
		}

		count = r.Pagination.TotalResults
		return nil
	}

	err = backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, step time.Duration) {
		logger.Error("failed-to-get-count", err, lager.Data{
			"backoff.step": step.String(),
		})
	})
	if err != nil {
		logger.Error("finally-failed-to-get-count", err)
		return 0, err
	}

	return count, err
}
