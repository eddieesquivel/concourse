package factory_test

import (
	"github.com/concourse/atc"
	"github.com/concourse/atc/scheduler/factory"
	"github.com/concourse/atc/testhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Factory Try Step", func() {
	var (
		buildFactory        factory.BuildFactory
		actualPlanFactory   atc.PlanFactory
		expectedPlanFactory atc.PlanFactory
	)

	BeforeEach(func() {
		actualPlanFactory = atc.NewPlanFactory(123)
		expectedPlanFactory = atc.NewPlanFactory(123)
		buildFactory = factory.NewBuildFactory("some-pipeline", actualPlanFactory)
	})

	Context("When there is a task wrapped in a try", func() {
		It("builds correctly", func() {
			actual, err := buildFactory.Create(atc.JobConfig{
				Plan: atc.PlanSequence{
					{
						Try: &atc.PlanConfig{
							Task: "first task",
						},
					},
					{
						Task: "second task",
					},
				},
			}, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			expected := expectedPlanFactory.NewPlan(atc.DoPlan{
				expectedPlanFactory.NewPlan(atc.TryPlan{
					Step: expectedPlanFactory.NewPlan(atc.TaskPlan{
						Name:     "first task",
						Pipeline: "some-pipeline",
					}),
				}),
				expectedPlanFactory.NewPlan(atc.TaskPlan{
					Name:     "second task",
					Pipeline: "some-pipeline",
				}),
			})

			Expect(actual).To(testhelpers.MatchPlan(expected))
		})
	})

	Context("When the try is in a hook", func() {
		It("builds correctly", func() {
			actual, err := buildFactory.Create(atc.JobConfig{
				Plan: atc.PlanSequence{
					{
						Task: "first task",
						Success: &atc.PlanConfig{
							Try: &atc.PlanConfig{
								Task: "second task",
							},
						},
					},
				},
			}, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			expected := expectedPlanFactory.NewPlan(atc.OnSuccessPlan{
				Step: expectedPlanFactory.NewPlan(atc.TaskPlan{
					Name:     "first task",
					Pipeline: "some-pipeline",
				}),
				Next: expectedPlanFactory.NewPlan(atc.TryPlan{
					Step: expectedPlanFactory.NewPlan(atc.TaskPlan{
						Name:     "second task",
						Pipeline: "some-pipeline",
					}),
				}),
			})

			Expect(actual).To(testhelpers.MatchPlan(expected))
		})
	})
})
