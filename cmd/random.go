package cmd

import (
	"math"
	"math/rand"

	"github.com/cloudfoundry-community/go-cfclient"
)

//go:generate counterfeiter math/rand.Source

// ChooseNumOrgAssignments returns a number of organization assignments sampled
// from the distribution.
//
// It does this by figuring out which "bucket" a randomly sampled user belongs to
// and returning the number of orgs assigned to the bucket.
func ChooseNumOrgAssignments(r *rand.Rand, distributions []UserOrgDistribution) uint {
	x := r.Float64()

	var cum float64
	for _, d := range distributions {
		if x > cum && x <= cum+d.PercentUsers {
			return uint(d.NumOrgs)
		}

		cum += d.PercentUsers
	}

	return 0
}

// ChooseNumSpaceAssignments returns a number of space assignments sampled
// from the distribution.
//
// It does this by figuring out which "bucket" a randomly sampled user belongs to
// and returning the number of spaces assigned to the bucket.
func ChooseNumSpaceAssignments(r *rand.Rand, distributions []UserSpaceDistribution) uint {
	x := r.Float64()

	var cum float64
	for _, d := range distributions {
		if x > cum && x <= cum+d.PercentUsers {
			return uint(d.NumSpaces)
		}

		cum += d.PercentUsers
	}

	return 0
}

// RandomlyChooseOrgs returns a contiguous window of size num of orgs out of the slice
//
// It does this by randomly choosing an index between 0 and (len orgs - window size)
func RandomlyChooseOrgs(r *rand.Rand, orgs []cfclient.Org, num uint) []cfclient.Org {
	maxIndex := int(math.Min(float64(len(orgs)-int(num)), float64(len(orgs))))

	idx := r.Intn(maxIndex)

	return orgs[idx:(idx + int(num))]
}

// RandomlyChooseSpaces returns a contiguous window of size num of spaces out of the slice
//
// It does this by randomly choosing an index between 0 and (len spaces - window size)
func RandomlyChooseSpaces(r *rand.Rand, spaces []cfclient.Space, num uint) []cfclient.Space {
	maxIndex := int(math.Min(float64(len(spaces)-int(num)), float64(len(spaces))))

	idx := r.Intn(maxIndex)

	return spaces[idx:(idx + int(num))]
}
