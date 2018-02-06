package cmd_test

import (
	"math/rand"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/perm-test/cmd/cmdfakes"

	. "github.com/pivotal-cf/perm-test/cmd"
)

var _ = Describe("Random", func() {
	Describe("ChooseNumOrgAssignments", func() {
		var (
			source *cmdfakes.FakeSource
			r      *rand.Rand

			distributions []UserOrgDistribution
		)

		Context("when provided with a distribution of % users/num orgs of 1%/500, 5%/50, and 94%/1", func() {
			BeforeEach(func() {
				source = new(cmdfakes.FakeSource)
				r = rand.New(source)

				distributions = []UserOrgDistribution{
					{PercentUsers: 0.01, NumOrgs: 500},
					{PercentUsers: 0.05, NumOrgs: 50},
					{PercentUsers: 0.94, NumOrgs: 1},
				}
			})

			It("should return 0 if no distributions are provided", func() {
				assignments := ChooseNumOrgAssignments(r, nil)
				Expect(assignments).To(Equal(uint(0)))
			})

			It("should return 500 if the random number generated is less than 0.01", func() {
				source.Int63Returns(int64(float64(0.009) * float64(1<<63)))
				Expect(r.Float64()).To(Equal(float64(0.009)))

				assignments := ChooseNumOrgAssignments(r, distributions)
				Expect(assignments).To(Equal(uint(500)))
			})

			It("should return 50 if the random number generated is between 0.01 and 0.06", func() {
				source.Int63Returns(int64(float64(0.05) * float64(1<<63)))
				Expect(r.Float64()).To(Equal(float64(0.05)))

				assignments := ChooseNumOrgAssignments(r, distributions)
				Expect(assignments).To(Equal(uint(50)))
			})

			It("should return 1 if the random number generated is greater than 0.06", func() {
				source.Int63Returns(int64(float64(0.5) * float64(1<<63)))
				Expect(r.Float64()).To(Equal(float64(0.5)))

				assignments := ChooseNumOrgAssignments(r, distributions)
				Expect(assignments).To(Equal(uint(1)))
			})
		})
	})
})
