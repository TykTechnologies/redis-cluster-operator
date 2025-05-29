package drc_crud_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	drctest "github.com/TykTechnologies/redis-cluster-operator/test/e2e/drc_crud/drc"
	"github.com/TykTechnologies/redis-cluster-operator/test/utils"
)

var (
	goredis *drctest.GoRedis
	dbsize  int64
	err     error
)

var _ = Describe("DistributedRedisCluster CRUD", Ordered, func() {
	It("should create a DistributedRedisCluster", func() {
		name := utils.RandString(8)
		password := utils.RandString(8)
		drc = drctest.NewDistributedRedisCluster(name, f.Namespace(), drctest.Redis6_2_6, f.PasswordName(), 3, 1)
		Ω(f.CreateRedisClusterPassword(f.PasswordName(), password)).Should(Succeed())
		Ω(f.CreateRedisCluster(drc)).Should(Succeed())

		// Sleep to allow DRC time to stabilize after creation
		time.Sleep(60 * time.Second)

		Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "15m", "10s").ShouldNot(HaveOccurred())

		// Sleep to give DRC more time to stabilize before further interactions
		time.Sleep(30 * time.Second)

		goredis = drctest.NewGoRedisClient(name, f.Namespace(), password)
		Expect(goredis.StuffingData(3, 50000)).NotTo(HaveOccurred())
		dbsize, err = goredis.DBSize()
		Expect(err).NotTo(HaveOccurred())
		f.Logf("%s DBSIZE: %d", name, dbsize)
	})

	Context("when the DistributedRedisCluster is created", func() {
		It("should change redis config for a DistributedRedisCluster", func() {
			drctest.ChangeDRCRedisConfig(drc)
			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "15m", "10s").ShouldNot(HaveOccurred())
			Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
		})
		It("should recover from accidentally deleting master pods", func() {
			// Sleep to give time for DRC to stabilize after any configuration changes
			time.Sleep(30 * time.Second)
			drctest.DeleteMasterPodForDRC(drc, f.Client)
			Eventually(drctest.IsDRCPodBeDeleted(f, drc), "5m", "10s").ShouldNot(HaveOccurred())

			// Sleep to allow time for recovery after pod deletion
			time.Sleep(60 * time.Second)

			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "15m", "10s").ShouldNot(HaveOccurred())

			// Sleep to allow time for DB client recovery
			time.Sleep(30 * time.Second)

			goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
			Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
		})
		It("should scale up a DistributedRedisCluster to 4", func() {
			// Sleep to give time for DRC to stabilize before scaling
			time.Sleep(30 * time.Second)
			drctest.ScaleUPDRC(drc, 4)
			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())

			// Sleep to allow time for cluster scaling
			time.Sleep(60 * time.Second)

			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "15m", "10s").ShouldNot(HaveOccurred())

			// Sleep to allow time for a DB client to stabilize after scaling
			time.Sleep(30 * time.Second)

			goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
			Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())

			// **print cluster nodes & status**
			goredis.PrintClusterState(4)

		})
		It("should scale up a DistributedRedisCluster to 5", func() {
			// Sleep to give time for DRC to stabilize before scaling
			time.Sleep(30 * time.Second)
			drctest.ScaleUPDRC(drc, 5)
			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())

			// Sleep to allow time for cluster scaling
			time.Sleep(60 * time.Second)

			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "15m", "10s").ShouldNot(HaveOccurred())

			// Sleep to allow time for a DB client to stabilize after scaling
			time.Sleep(30 * time.Second)

			goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
			Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())

			// **print cluster nodes & status**
			goredis.PrintClusterState(5)
		})
		Context("when the scale up succeeded", func() {
			It("should scale down a DistributedRedisCluster", func() {
				// Sleep to give time for DRC to stabilize before scaling down
				time.Sleep(30 * time.Second)

				drctest.ScaleUPDown(drc)
				Ω(f.UpdateRedisCluster(drc)).Should(Succeed())

				// Sleep to allow time for cluster scaling down
				time.Sleep(60 * time.Second)

				Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "15m", "10s").ShouldNot(HaveOccurred())

				// Sleep to allow time for the DB client to stabilize after scaling down
				time.Sleep(30 * time.Second)

				goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
				Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())

				goredis.PrintClusterState(3)
			})
		})
		It("should reset the DistributedRedisCluster password", func() {
			// Sleep to allow time for DRC to stabilize before password reset
			time.Sleep(30 * time.Second)
			newPassword := utils.RandString(8)
			Ω(f.CreateRedisClusterPassword(f.NewPasswordName(), newPassword)).Should(Succeed())
			drctest.ResetPassword(drc, f.NewPasswordName())
			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())

			// Sleep to allow time for DRC to stabilize after password reset
			time.Sleep(60 * time.Second)

			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "15m", "10s").ShouldNot(HaveOccurred())

			// Sleep to allow time for the DB client to stabilize after password reset
			time.Sleep(30 * time.Second)

			goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), newPassword)
			Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
		})
		It("should update the DistributedRedisCluster minor version", func() {
			// Sleep to give time for DRC to stabilize before the update
			time.Sleep(30 * time.Second)
			drctest.RollingUpdateDRC(drc)
			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())

			// Sleep to allow time for the DRC to stabilize after the update
			time.Sleep(60 * time.Second)

			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "15m", "10s").ShouldNot(HaveOccurred())

			// Sleep to allow time for the DB client to stabilize after update
			time.Sleep(30 * time.Second)

			goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
			Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
		})
		Context("when the DistributedRedisCluster has passed all tests", func() {
			It("should create a RedisClusterCleanup", func() {
				time.Sleep(30 * time.Second)
				name := utils.RandString(8)
				drccleanup := drctest.NewRedisClusterCleanup(name, drc)
				Ω(f.CreateRedisClusterCleaup(drccleanup)).Should(Succeed())
				// Sleep to allow DRC time to stabilize after creation
				time.Sleep(60 * time.Second)

				Eventually(drctest.IsRedisClusterCleanupProperly(f, drccleanup), "15m", "10s").ShouldNot(HaveOccurred())

				time.Sleep(30 * time.Second)

				goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
				// we keep non exipired keys and the keys with skip pattern
				Expect(drctest.IsDBSizeConsistent(35000, goredis)).NotTo(HaveOccurred())
			})
		})

	})
})
