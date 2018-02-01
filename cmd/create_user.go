package cmd

import (
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/go-cfclient"
)

func CreateUser(logger lager.Logger, cfClient *cfclient.Client, userGUID string) (*cfclient.User, error) {
	userRequest := cfclient.UserRequest{
		Guid: userGUID,
	}
	logger.Debug("creating-user", lager.Data{
		"guid": userGUID,
	})
	user, err := cfClient.CreateUser(userRequest)
	if err != nil {
		logger.Error("failed-to-create-cf-user", err)
		return nil, err
	}

	return &user, nil
}
