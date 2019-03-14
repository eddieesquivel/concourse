package db_test

import (
	"time"

	"github.com/cloudfoundry/bosh-cli/director/template"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/creds"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/db/lock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resource Config Scope", func() {
	var resourceScope db.ResourceConfigScope

	BeforeEach(func() {
		setupTx, err := dbConn.Begin()
		Expect(err).ToNot(HaveOccurred())

		brt := db.BaseResourceType{
			Name: "some-type",
		}

		_, err = brt.FindOrCreate(setupTx, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(setupTx.Commit()).To(Succeed())

		pipeline, _, err := defaultTeam.SavePipeline("scope-pipeline", atc.Config{
			Resources: atc.ResourceConfigs{
				{
					Name: "some-resource",
					Type: "some-type",
					Source: atc.Source{
						"some": "source",
					},
				},
			},
		}, db.ConfigVersion(0), db.PipelineUnpaused)
		Expect(err).NotTo(HaveOccurred())

		resource, found, err := pipeline.Resource("some-resource")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())

		resourceScope, err = resource.SetResourceConfig(logger, atc.Source{"some": "source"}, creds.VersionedResourceTypes{})
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("LatestVersions", func() {
		var (
			resourceConfig db.ResourceConfig
			latestCV       []db.ResourceVersion
		)

		Context("when the resource config exists", func() {
			BeforeEach(func() {
				setupTx, err := dbConn.Begin()
				Expect(err).ToNot(HaveOccurred())

				brt := db.BaseResourceType{
					Name: "some-type",
				}
				_, err = brt.FindOrCreate(setupTx)
				Expect(err).NotTo(HaveOccurred())
				Expect(setupTx.Commit()).To(Succeed())

				resourceConfigFactory := db.NewResourceConfigFactory(dbConn, lockFactory)
				resourceConfig, err = resourceConfigFactory.FindOrCreateResourceConfig(logger, "some-type", atc.Source{"source-config": "some-value"}, creds.VersionedResourceTypes{})
				Expect(err).ToNot(HaveOccurred())

				saveVersions(resourceConfig, []atc.SpaceVersion{
					atc.SpaceVersion{
						Version: atc.Version{"ref": "v1"},
						Space:   atc.Space("space"),
					},
					atc.SpaceVersion{
						Version: atc.Version{"ref": "v3"},
						Space:   atc.Space("space"),
					},
				})

				err = resourceConfig.SaveSpaceLatestVersion(atc.Space("space"), atc.Version{"ref": "v3"})
				Expect(err).ToNot(HaveOccurred())

				latestCV, err = resourceConfig.LatestVersions()
				Expect(err).ToNot(HaveOccurred())
			})

			It("gets latest version of resource", func() {
				Expect(latestCV).To(HaveLen(1))
				Expect(latestCV[0].Version()).To(Equal(db.Version{"ref": "v3"}))
				Expect(latestCV[0].CheckOrder()).To(Equal(2))
			})
		})
	})

	Describe("FindVersion", func() {
		var (
			resourceConfig db.ResourceConfig
			latestCV       db.ResourceVersion
			found          bool
		)

		BeforeEach(func() {
			setupTx, err := dbConn.Begin()
			Expect(err).ToNot(HaveOccurred())

			brt := db.BaseResourceType{
				Name: "some-type",
			}

			resourceConfig, err = resourceConfigFactory.FindOrCreateResourceConfig(logger, "some-type", atc.Source{"source-config": "some-value"}, creds.VersionedResourceTypes{})

			saveVersions(resourceConfig, []atc.SpaceVersion{
				atc.SpaceVersion{
					Version: atc.Version{"ref": "v1"},
					Space:   atc.Space("space"),
				},
				atc.SpaceVersion{
					Version: atc.Version{"ref": "v3"},
					Space:   atc.Space("space"),
				},
			})
		})

		Context("when the version exists", func() {
			BeforeEach(func() {
				var err error
				latestCV, found, err = resourceConfig.FindVersion(atc.Space("space"), atc.Version{"ref": "v1"})
				Expect(err).ToNot(HaveOccurred())
			})

			It("gets the version of resource", func() {
				Expect(found).To(BeTrue())

				Expect(latestCV.ResourceConfig().ID()).To(Equal(resourceConfig.ID()))
				Expect(latestCV.Version()).To(Equal(db.Version{"ref": "v1"}))
				Expect(latestCV.CheckOrder()).To(Equal(1))
			})
		})

		Context("when the version does not exist", func() {
			BeforeEach(func() {
				var err error
				latestCV, found, err = resourceConfig.FindVersion(atc.Space("space"), atc.Version{"ref": "v2"})
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not get the version of resource", func() {
				Expect(found).To(BeFalse())
			})
		})

		Context("when the space does not exist", func() {
			BeforeEach(func() {
				var err error
				latestCV, found, err = resourceConfig.FindVersion(atc.Space("non-existant"), atc.Version{"ref": "v2"})
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not get the version of resource", func() {
				Expect(found).To(BeFalse())
			})
		})
	})

	Describe("FindUncheckedVersion", func() {
		var (
			resourceConfig db.ResourceConfig
			latestCV       db.ResourceVersion
			found          bool
		)

		BeforeEach(func() {
			setupTx, err := dbConn.Begin()
			Expect(err).ToNot(HaveOccurred())

			brt := db.BaseResourceType{
				Name: "some-type",
			}
			_, err = brt.FindOrCreate(setupTx)
			Expect(err).NotTo(HaveOccurred())
			Expect(setupTx.Commit()).To(Succeed())

			resourceConfig, err = resourceConfigFactory.FindOrCreateResourceConfig(logger, "some-type", atc.Source{"source-config": "some-value"}, creds.VersionedResourceTypes{})
			Expect(err).ToNot(HaveOccurred())

			err = resourceConfig.SaveSpace(atc.Space("space"))
			Expect(err).ToNot(HaveOccurred())

			_, err = resourceConfig.SaveUncheckedVersion(atc.Space("space"), atc.Version{"ref": "v1"}, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			var err error
			latestCV, found, err = resourceConfig.FindUncheckedVersion(atc.Space("space"), atc.Version{"ref": "v1"})
			Expect(err).ToNot(HaveOccurred())
		})

		It("gets the version of resource", func() {
			Expect(found).To(BeTrue())

			Expect(latestCV.ResourceConfig().ID()).To(Equal(resourceConfig.ID()))
			Expect(latestCV.Version()).To(Equal(db.Version{"ref": "v1"}))
			Expect(latestCV.CheckOrder()).To(Equal(0))
		})
	})

	Describe("SaveDefaultSpace", func() {
		var (
			resourceConfig  db.ResourceConfig
			defaultSpace    string
			defaultSpaceErr error
		)

		BeforeEach(func() {
			setupTx, err := dbConn.Begin()
			Expect(err).ToNot(HaveOccurred())

			brt := db.BaseResourceType{
				Name: "some-type",
			}
			_, err = brt.FindOrCreate(setupTx)
			Expect(err).NotTo(HaveOccurred())
			Expect(setupTx.Commit()).To(Succeed())

			resourceConfigFactory := db.NewResourceConfigFactory(dbConn, lockFactory)
			resourceConfig, err = resourceConfigFactory.FindOrCreateResourceConfig(logger, "some-type", atc.Source{"source-config": "some-value"}, creds.VersionedResourceTypes{})
			Expect(err).ToNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			defaultSpaceErr = resourceConfig.SaveDefaultSpace(atc.Space(defaultSpace))
		})

		Context("when the space exists", func() {
			BeforeEach(func() {
				err := resourceConfig.SaveSpace(atc.Space("space"))
				Expect(err).ToNot(HaveOccurred())

				defaultSpace = "space"
			})

			It("saves the default space", func() {
				Expect(defaultSpaceErr).ToNot(HaveOccurred())

				resourceConfig, err := resourceConfigFactory.FindOrCreateResourceConfig(logger, "some-type", atc.Source{"source-config": "some-value"}, creds.VersionedResourceTypes{})
				Expect(err).ToNot(HaveOccurred())
				Expect(resourceConfig.DefaultSpace()).To(Equal(atc.Space("space")))
			})
		})
	})

	Describe("SavePartialVersion/SaveSpace", func() {
		var (
			resourceConfig db.ResourceConfig
			spaceVersion   atc.SpaceVersion
		)

		BeforeEach(func() {
			setupTx, err := dbConn.Begin()
			Expect(err).ToNot(HaveOccurred())

			brt := db.BaseResourceType{
				Name: "some-type",
			}
			_, err = brt.FindOrCreate(setupTx)
			Expect(err).NotTo(HaveOccurred())
			Expect(setupTx.Commit()).To(Succeed())

			resourceConfigFactory := db.NewResourceConfigFactory(dbConn, lockFactory)
			resourceConfig, err = resourceConfigFactory.FindOrCreateResourceConfig(logger, "some-type", atc.Source{"source-config": "some-value"}, creds.VersionedResourceTypes{})
			Expect(err).ToNot(HaveOccurred())

			spaceVersion = atc.SpaceVersion{
				Space:   "space",
				Version: atc.Version{"some": "version"},
				Metadata: atc.Metadata{
					atc.MetadataField{
						Name:  "some",
						Value: "metadata",
					},
				},
			}
		})

		It("saves the version if the space exists", func() {
			saveVersions(resourceConfig, []atc.SpaceVersion{spaceVersion})

			err := resourceConfig.SaveSpaceLatestVersion(atc.Space("space"), atc.Version{"some": "version"})
			Expect(err).ToNot(HaveOccurred())

			latestVR, err := resourceConfig.LatestVersions()
			Expect(err).ToNot(HaveOccurred())
			Expect(latestVR).ToNot(BeEmpty())
			Expect(latestVR[0].Version()).To(Equal(db.Version{"some": "version"}))
			Expect(latestVR[0].CheckOrder()).To(Equal(1))
		})

		Context("when the space does not exist", func() {
			BeforeEach(func() {
				spaceVersion = atc.SpaceVersion{
					Space:   "unknown-space",
					Version: atc.Version{"some": "version"},
					Metadata: atc.Metadata{
						atc.MetadataField{
							Name:  "some",
							Value: "metadata",
						},
					},
				}
			})

			It("does not save the version", func() {
				err := resourceConfig.SavePartialVersion(spaceVersion.Space, spaceVersion.Version, spaceVersion.Metadata)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when saving multiple versions", func() {
			It("ensures versions have the correct check_order", func() {
				originalVersionSlice := []atc.SpaceVersion{
					{
						Space:   atc.Space("space"),
						Version: atc.Version{"ref": "v1"},
					},
					{
						Space:   atc.Space("space"),
						Version: atc.Version{"ref": "v3"},
					},
				}

				saveVersions(resourceConfig, originalVersionSlice)

				err := resourceConfig.SaveSpaceLatestVersion(atc.Space("space"), atc.Version{"ref": "v3"})
				Expect(err).ToNot(HaveOccurred())

				latestVR, err := resourceConfig.LatestVersions()
				Expect(err).ToNot(HaveOccurred())
				Expect(latestVR[0].Version()).To(Equal(db.Version{"ref": "v3"}))
				Expect(latestVR[0].CheckOrder()).To(Equal(2))

				pretendCheckResults := []atc.SpaceVersion{
					{
						Space:   atc.Space("space"),
						Version: atc.Version{"ref": "v2"},
					},
					{
						Space:   atc.Space("space"),
						Version: atc.Version{"ref": "v3"},
					},
				}

				saveVersions(resourceConfig, pretendCheckResults)

				err = resourceConfig.SaveSpaceLatestVersion(atc.Space("space"), atc.Version{"ref": "v3"})
				Expect(err).ToNot(HaveOccurred())

				latestVR, err = resourceConfig.LatestVersions()
				Expect(err).ToNot(HaveOccurred())
				Expect(latestVR[0].Version()).To(Equal(db.Version{"ref": "v3"}))
				Expect(latestVR[0].CheckOrder()).To(Equal(4))
			})
		})

		Context("when the versions already exists", func() {
			var newVersionSlice []atc.SpaceVersion

			BeforeEach(func() {
				newVersionSlice = []atc.SpaceVersion{
					{
						Space:   atc.Space("space"),
						Version: atc.Version{"ref": "v1"},
					},
					{
						Space:   atc.Space("space"),
						Version: atc.Version{"ref": "v3"},
					},
				}
			})

			It("does not change the check order", func() {
				saveVersions(resourceConfig, newVersionSlice)

				err := resourceConfig.SaveSpaceLatestVersion(atc.Space("space"), atc.Version{"ref": "v3"})
				Expect(err).ToNot(HaveOccurred())

				latestVR, err := resourceConfig.LatestVersions()
				Expect(err).ToNot(HaveOccurred())

				Expect(latestVR[0].Version()).To(Equal(db.Version{"ref": "v3"}))
				Expect(latestVR[0].CheckOrder()).To(Equal(2))
			})
		})
	})

	Describe("SaveSpaceLatestVersion/LatestVersions", func() {
		var (
			resourceConfig db.ResourceConfig
			spaceVersion   atc.SpaceVersion
			spaceVersion2  atc.SpaceVersion
		)

		BeforeEach(func() {
			setupTx, err := dbConn.Begin()
			Expect(err).ToNot(HaveOccurred())

			brt := db.BaseResourceType{
				Name: "some-type",
			}
			_, err = brt.FindOrCreate(setupTx)
			Expect(err).NotTo(HaveOccurred())
			Expect(setupTx.Commit()).To(Succeed())

			resourceConfigFactory := db.NewResourceConfigFactory(dbConn, lockFactory)
			resourceConfig, err = resourceConfigFactory.FindOrCreateResourceConfig(logger, "some-type", atc.Source{"source-config": "some-value"}, creds.VersionedResourceTypes{})
			Expect(err).ToNot(HaveOccurred())

			err = resourceConfig.SaveSpace(atc.Space("space"))
			Expect(err).ToNot(HaveOccurred())

			otherSpaceVersion := atc.SpaceVersion{
				Space:   "space",
				Version: atc.Version{"some": "other-version"},
				Metadata: atc.Metadata{
					atc.MetadataField{
						Name:  "some",
						Value: "metadata",
					},
				},
			}

			spaceVersion = atc.SpaceVersion{
				Space:   "space",
				Version: atc.Version{"some": "version"},
				Metadata: atc.Metadata{
					atc.MetadataField{
						Name:  "some",
						Value: "metadata",
					},
				},
			}

			saveVersions(resourceConfig, []atc.SpaceVersion{otherSpaceVersion, spaceVersion})

			err = resourceConfig.SaveSpace(atc.Space("space2"))
			Expect(err).ToNot(HaveOccurred())

			spaceVersion2 = atc.SpaceVersion{
				Space:   "space2",
				Version: atc.Version{"some": "version2"},
				Metadata: atc.Metadata{
					atc.MetadataField{
						Name:  "some",
						Value: "metadata",
					},
				},
			}

			saveVersions(resourceConfig, []atc.SpaceVersion{spaceVersion2})
		})

		Context("when the version exists", func() {
			BeforeEach(func() {
				err := resourceConfig.SaveSpaceLatestVersion(spaceVersion.Space, spaceVersion.Version)
				Expect(err).ToNot(HaveOccurred())

				err = resourceConfig.SaveSpaceLatestVersion(spaceVersion2.Space, spaceVersion2.Version)
				Expect(err).ToNot(HaveOccurred())
			})

			It("saves the version into the space", func() {
				latestVersions, err := resourceConfig.LatestVersions()
				Expect(err).ToNot(HaveOccurred())
				Expect(latestVersions).To(HaveLen(2))
				Expect(latestVersions[0].Version()).To(Equal(db.Version(spaceVersion.Version)))
				Expect(latestVersions[1].Version()).To(Equal(db.Version(spaceVersion2.Version)))
			})
		})
	})

	Describe("UpdateLastChecked", func() {
		var (
			someResource        db.Resource
			resourceConfigScope db.ResourceConfigScope
		)

		BeforeEach(func() {
			var err error
			var found bool

			someResource, found, err = defaultPipeline.Resource("some-resource")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())

			pipelineResourceTypes, err := defaultPipeline.ResourceTypes()
			Expect(err).ToNot(HaveOccurred())

			resourceConfigScope, err = someResource.SetResourceConfig(
				logger,
				someResource.Source(),
				creds.NewVersionedResourceTypes(template.StaticVariables{}, pipelineResourceTypes.Deserialize()),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when there has not been a check", func() {
			It("should update the last checked", func() {
				updated, err := resourceConfigScope.UpdateLastChecked(1*time.Second, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(updated).To(BeTrue())
			})

			Context("when immediate", func() {
				It("should update the last checked", func() {
					updated, err := resourceConfigScope.UpdateLastChecked(1*time.Second, true)
					Expect(err).ToNot(HaveOccurred())
					Expect(updated).To(BeTrue())
				})
			})
		})

		Context("when there has been a check recently", func() {
			BeforeEach(func() {
				updated, err := resourceConfigScope.UpdateLastChecked(1*time.Second, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(updated).To(BeTrue())
			})

			Context("when not immediate", func() {
				It("does not update the last checked", func() {
					updated, err := resourceConfigScope.UpdateLastChecked(1*time.Second, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(updated).To(BeFalse())
				})

				It("updates the last checked and stops others from periodically updating at the same time", func() {
					Consistently(func() bool {
						updated, err := resourceConfigScope.UpdateLastChecked(1*time.Second, false)
						Expect(err).ToNot(HaveOccurred())

						return updated
					}, time.Second, 100*time.Millisecond).Should(BeFalse())

					time.Sleep(time.Second)

					updated, err := resourceConfigScope.UpdateLastChecked(1*time.Second, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(updated).To(BeTrue())
				})
			})

			Context("when it is immediate", func() {
				It("updates the last checked and stops others from updating too", func() {
					updated, err := resourceConfigScope.UpdateLastChecked(1*time.Second, true)
					Expect(err).ToNot(HaveOccurred())
					Expect(updated).To(BeTrue())
				})
			})
		})
	})

	Describe("AcquireResourceCheckingLock", func() {
		var (
			someResource        db.Resource
			resourceConfigScope db.ResourceConfigScope
		)

		BeforeEach(func() {
			var err error
			var found bool

			someResource, found, err = defaultPipeline.Resource("some-resource")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())

			pipelineResourceTypes, err := defaultPipeline.ResourceTypes()
			Expect(err).ToNot(HaveOccurred())

			resourceConfigScope, err = someResource.SetResourceConfig(
				logger,
				someResource.Source(),
				creds.NewVersionedResourceTypes(template.StaticVariables{}, pipelineResourceTypes.Deserialize()),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when there has been a check recently", func() {
			var lock lock.Lock
			var err error

			BeforeEach(func() {
				var err error
				var acquired bool
				lock, acquired, err = resourceConfigScope.AcquireResourceCheckingLock(logger, 1*time.Second)
				Expect(err).ToNot(HaveOccurred())
				Expect(acquired).To(BeTrue())
			})

			It("does not get the lock", func() {
				_, acquired, err := resourceConfigScope.AcquireResourceCheckingLock(logger, 1*time.Second)
				Expect(err).ToNot(HaveOccurred())
				Expect(acquired).To(BeFalse())
			})

			Context("and the lock gets released", func() {
				BeforeEach(func() {
					err = lock.Release()
					Expect(err).ToNot(HaveOccurred())
				})

				It("gets the lock", func() {
					lock, acquired, err := resourceConfigScope.AcquireResourceCheckingLock(logger, 1*time.Second)
					Expect(err).ToNot(HaveOccurred())
					Expect(acquired).To(BeTrue())

					err = lock.Release()
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("when there has not been a check recently", func() {
			It("gets and keeps the lock and stops others from periodically getting it", func() {
				lock, acquired, err := resourceConfigScope.AcquireResourceCheckingLock(logger, 1*time.Second)
				Expect(err).ToNot(HaveOccurred())
				Expect(acquired).To(BeTrue())

				Consistently(func() bool {
					_, acquired, err = resourceConfigScope.AcquireResourceCheckingLock(logger, 1*time.Second)
					Expect(err).ToNot(HaveOccurred())

					return acquired
				}, 1500*time.Millisecond, 100*time.Millisecond).Should(BeFalse())

				err = lock.Release()
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(time.Second)

				lock, acquired, err = resourceConfigScope.AcquireResourceCheckingLock(logger, 1*time.Second)
				Expect(err).ToNot(HaveOccurred())
				Expect(acquired).To(BeTrue())

				err = lock.Release()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

func saveVersions(resourceConfig db.ResourceConfig, versions []atc.SpaceVersion) {
	for _, version := range versions {
		err := resourceConfig.SaveSpace(version.Space)
		Expect(err).ToNot(HaveOccurred())

		err = resourceConfig.SavePartialVersion(version.Space, version.Version, version.Metadata)
		Expect(err).ToNot(HaveOccurred())
	}

	err := resourceConfig.FinishSavingVersions()
	Expect(err).ToNot(HaveOccurred())
}