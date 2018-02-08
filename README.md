# perm-test

perm-test is used to performance test a large dataset of variable users/orgs/spaces/apps on Cloud Foundry, integrated with the new CF Permissions component.

It does this by first seeding the CF+Perm database with the dataset, and follows it up by testing the load (i.e. Apache Benchmark, etc.).

## Usage

### Set up a user in the UAA with a known username/password

```
https_proxy=$BOSH_ALL_PROXY credhub login -u $CREDHUB_USER -p $CREDHUB_PASSWORD
uaa_client_admin_secret="$(https_proxy="${BOSH_ALL_PROXY}" credhub get -n "/bosh-${ENVIRONMENT_NAME}/cf/uaa_admin_client_secret" -j | jq -r .value)"
uaac token client get admin --secret "${uaa_client_admin_secret}"
uaac user add user --emails user --password password
```

After this, do `uaac users` and note the GUID of the generated user

### Create a config file

Create a yml config file for the experiment you want to run. It must be filled in with information about the environment you are seeding

```
log_level: info

cloud_controller:
  client_id:
  client_secret:
  url:

test_data:
  spaces_per_org_count: 10
  apps_per_space_count: 10
  test_environment:
    user_guid: <uaac_user_guid>
    org_count: 400
  external_environment:
    org_count: 4000
    user_count: 1000

    # percent_users   MUST sum to 1
    # num_spaces      NEED NOT sum to total spaces, MUST NOT be greater than total spaces
    # num_orgs        NEED NOT sum to total orgs,   MUST NOT be greater than total orgs
    # Max num_spaces  SHOULD be less than test_environment spaces
    # Max num_orgs    SHOULD be less than test_environment orgs
    user_org_distribution:
    - percent_users: .01
      num_orgs: 300
    - percent_users: .05
      num_orgs: 50
    - percent_users: .94
      num_orgs: 1

    user_space_distribution:
    - percent_users: .01
      num_spaces: 3000
    - percent_users: .05
      num_spaces: 500
    - percent_users: .5
      num_spaces: 5
    - percent_users: .44
      num_spaces: 1
```

### Seed Data

```
go install github.com/pivotal-cf/perm-test/cmd/loaddata
time loaddata <path/to/config.yml>
```


### Run Experiments

```
ENVIRONMENT_NAME=cleopatra CF_USERNAME=user CF_PASSWORD=password ./scripts/experiment1.sh
```

## Caveats!

If you *DID NOT* create a user through uaa and instead through cloud controller/migration script first,
you will not be able to log in as that user.
You must use uaac to generate a new user with a known username/password (we suggest `user`/`password`),
then grab the UAA guid that corresponds to that, and then modify the guid in the cloud_controller database to that
e.g. `use cloud_controller; update users set guid="<new-uaa-guid>" where guid="<old-cloud-controller-guid>";`
