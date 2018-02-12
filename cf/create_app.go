package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cenkalti/backoff"
	"github.com/cloudfoundry-community/go-cfclient"
)

// CreateAppIfNotExists creates an app in CloudFoundry using the V3 API
// It uses an exponential backoff strategy, returning early if it successfully creates
// an app or the app already exists
func CreateAppIfNotExists(logger lager.Logger, cfClient *cfclient.Client, name string, spaceGUID string) error {
	req := &CreateAppRequestBody{
		Name: name,
		Relationships: SpaceRelationship{
			Space: Space{
				Data: Data{GUID: spaceGUID},
			},
		},
	}

	operation := func() error {
		b := bytes.NewBuffer(nil)
		err := json.NewEncoder(b).Encode(req)
		if err != nil {
			return err
		}

		r := cfClient.NewRequestWithBody("POST", "/v3/apps", b)
		resp, err := cfClient.DoRequest(r)

		switch e := err.(type) {
		case nil:
			if resp.StatusCode != http.StatusCreated {
				err = fmt.Errorf("Incorrect status code (%d)", resp.StatusCode)

			}
			return err

		case cfclient.V3CloudFoundryErrors:
			if len(e.Errors) == 0 {
				return err
			}

			//
			// Setting a Timeout on the HttpClient, there is a risk of
			// the request succeeding, but being cancelled on the client side
			// This causes the app to be created, but the backoff thinks it failed
			// due to timeout. When it tries the operation a second time, it fails
			// with "name must be unique in space" error, because it already exists.
			//
			// To get around this, we return nil if we detect this case, which causes
			// the backoff function to stop retrying.
			//
			for _, cfError := range e.Errors {
				if cfError.Detail == "name must be unique in space" {
					return nil
				}
			}

			return err
		default:
			return err
		}
	}

	err := backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, step time.Duration) {
		logger.Error("failed-to-create-app", err, lager.Data{
			"backoff.step": step.String(),
		})
	})

	if err != nil {
		logger.Error("finally-failed-to-create-app", err)
	}

	return err
}

type SpaceGUID string

type CreateAppRequestBody struct {
	Name          string            `json:"name"`
	Relationships SpaceRelationship `json:"relationships"`
}

type SpaceRelationship struct {
	Space Space `json:"space"`
}

type Space struct {
	Data Data `json:"data"`
}

type Data struct {
	GUID string `json:"guid"`
}
