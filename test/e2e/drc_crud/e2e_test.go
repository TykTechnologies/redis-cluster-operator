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

var _ = Describe("DistributedRedisCluster CRUD", func() {
	It("should create a DistributedRedisCluster", func() {
		name := utils.RandString(8)
		password := utils.RandString(8)
		drc = drctest.NewDistributedRedisCluster(name, f.Namespace(), drctest.Redis5_0_4, f.PasswordName(), 3, 1)
		Ω(f.CreateRedisClusterPassword(f.PasswordName(), password)).Should(Succeed())
		Ω(f.CreateRedisCluster(drc)).Should(Succeed())
		Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
		//goredis = drctest.NewGoRedisClient(name, f.Namespace(), password)
		//Expect(goredis.StuffingData(10, 300000)).NotTo(HaveOccurred())
		//dbsize, err = goredis.DBSize()
		//Expect(err).NotTo(HaveOccurred())
		//f.Logf("%s DBSIZE: %d", name, dbsize)
	})

	Context("when the DistributedRedisCluster is created", func() {
		It("should change redis config for a DistributedRedisCluster", func() {
			drctest.ChangeDRCRedisConfig(drc)
			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
			//Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
		})
		It("should recover from accidentally deleting master pods", func() {
			drctest.DeleteMasterPodForDRC(drc, f.Client)
			Eventually(drctest.IsDRCPodBeDeleted(f, drc), "5m", "10s").ShouldNot(HaveOccurred())
			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
			//goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
			//Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
		})
		It("should scale up a DistributedRedisCluster", func() {
			drctest.ScaleUPDRC(drc)
			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
			//goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
			//Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
		})
		Context("when the scale up succeeded", func() {
			It("should scale down a DistributedRedisCluster", func() {
				drctest.ScaleUPDown(drc)
				Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
				Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
				//goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
				//Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
			})
		})
		It("should reset the DistributedRedisCluster password", func() {
			newPassword := utils.RandString(8)
			Ω(f.CreateRedisClusterPassword(f.NewPasswordName(), newPassword)).Should(Succeed())
			drctest.ResetPassword(drc, f.NewPasswordName())
			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
			time.Sleep(5 * time.Second)
			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
			//goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), newPassword)
			//Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
		})
		It("should update the DistributedRedisCluster minor version", func() {
			drctest.RollingUpdateDRC(drc)
			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
			Eventually(drctest.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
			//goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
			//Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
		})
	})
})
