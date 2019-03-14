package resource_test

import (
	"testing"

	"github.com/concourse/concourse/atc/resource"
	"github.com/concourse/concourse/atc/worker/workerfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	workerClient  *workerfakes.FakeClient
	fakeContainer *workerfakes.FakeContainer

	unversionedResource resource.UnversionedResource
)

var _ = BeforeEach(func() {
	workerClient = new(workerfakes.FakeClient)

	fakeContainer = new(workerfakes.FakeContainer)

	unversionedResource = resource.NewUnversionedResource(fakeContainer)
})

func TestResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resource Suite")
}