package cf_test

import (
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/cloudfoundry-community/go-cfclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/pivotal-cf/perm-test/cf"
)

var _ = Describe("Count", func() {
	var (
		server *ghttp.Server

		cfClient *cfclient.Client
		logger   *lagertest.TestLogger
	)

	BeforeEach(func() {
		server = ghttp.NewServer()

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/v2/info"),
			ghttp.RespondWith(200, "{}", nil),
		))

		var err error
		cfClient, err = cfclient.NewClient(&cfclient.Config{
			ApiAddress: "http://" + server.Addr(),
			Token:      "foobar",
		})

		Expect(err).NotTo(HaveOccurred())

		logger = lagertest.NewTestLogger("count")
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("OrgCount", func() {
		It("asks for /v3/organizations and parses the count", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v3/organizations"),
				ghttp.RespondWith(200, `{"pagination": {"total_results": 42}}`, nil),
			))

			count, err := OrgCount(logger, cfClient)
			Expect(err).NotTo(HaveOccurred())

			Expect(count).To(Equal(42))
		})
	})

	Describe("SpaceCount", func() {
		It("asks for /v3/spaces and parses the count", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v3/spaces"),
				ghttp.RespondWith(200, `{"pagination": {"total_results": 42}}`, nil),
			))

			count, err := SpaceCount(logger, cfClient)
			Expect(err).NotTo(HaveOccurred())

			Expect(count).To(Equal(42))
		})
	})

	Describe("UserCount", func() {
		It("asks for /v2/users and parses the count", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v2/users"),
				ghttp.RespondWith(200, `{"total_results": 42}`, nil),
			))

			count, err := UserCount(logger, cfClient)
			Expect(err).NotTo(HaveOccurred())

			Expect(count).To(Equal(42))
		})
	})
})
