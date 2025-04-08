/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
...
*/

package drc_operator_test

import (
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TykTechnologies/redis-cluster-operator/test/utils"
)

//var (
//	f       *drc2.Framework
//	drc     *redisv1alpha1.DistributedRedisCluster
//	goredis *drc2.GoRedis
//	dbsize  int64
//	err     error
//)

var _ = Describe("Operator Controller", func() {
	It("should have the controller-manager pod running", func() {
		var controllerPodName string
		verifyControllerUp := func() error {
			// Get pod name

			cmd := exec.Command("kubectl", "get",
				"pods", "-l", "control-plane=controller-manager",
				"-o", "go-template={{ range .items }}"+
					"{{ if not .metadata.deletionTimestamp }}"+
					"{{ .metadata.name }}"+
					"{{ \"\\n\" }}{{ end }}{{ end }}",
				"-n", namespace,
			)

			podOutput, err := utils.Run(cmd)
			ExpectWithOffset(2, err).NotTo(HaveOccurred())
			podNames := utils.GetNonEmptyLines(string(podOutput))
			if len(podNames) != 1 {
				return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
			}
			controllerPodName = podNames[0]
			ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

			// Validate pod status
			cmd = exec.Command("kubectl", "get",
				"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
				"-n", namespace,
			)
			status, err := utils.Run(cmd)
			ExpectWithOffset(2, err).NotTo(HaveOccurred())
			if string(status) != "Running" {
				return fmt.Errorf("controller pod in %s status", status)
			}
			return nil
		}
		Eventually(verifyControllerUp, time.Minute, time.Second).Should(Succeed())
	})
})

//Describe("DistributedRedisCluster CRUD", func() {
//
//	BeforeAll(func() {
//		f = drc2.NewFramework("test")
//		if err := f.BeforeEach(); err != nil {
//			f.Failf("Framework BeforeEach error: %s", err.Error())
//		}
//	})
//
//	AfterAll(func() {
//		if drc != nil {
//			if err := f.DeleteRedisCluster(drc); err != nil {
//				f.Logf("deleting DistributedRedisCluster error: %s", err.Error())
//			}
//		}
//		if err := f.AfterEach(); err != nil {
//			f.Failf("Framework AfterEach error: %s", err.Error())
//		}
//	})
//
//	It("should create a DistributedRedisCluster", func() {
//		name := utils.RandString(8)
//		password := utils.RandString(8)
//		drc = drc2.NewDistributedRedisCluster(name, f.Namespace(), drc2.Redis5_0_4, f.PasswordName(), 3, 1)
//		Ω(f.CreateRedisClusterPassword(f.PasswordName(), password)).Should(Succeed())
//		Ω(f.CreateRedisCluster(drc)).Should(Succeed())
//		Eventually(drc2.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
//		// goredis = drctest.NewGoRedisClient(name, f.Namespace(), password)
//		// Expect(goredis.StuffingData(10, 300000)).NotTo(HaveOccurred())
//		// dbsize, err = goredis.DBSize()
//		// Expect(err).NotTo(HaveOccurred())
//		// f.Logf("%s DBSIZE: %d", name, dbsize)
//	})
//
//	Context("when the DistributedRedisCluster is created", func() {
//		It("should change redis config for a DistributedRedisCluster", func() {
//			drc2.ChangeDRCRedisConfig(drc)
//			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
//			Eventually(drc2.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
//			// Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
//		})
//		It("should recover from accidentally deleting master pods", func() {
//			drc2.DeleteMasterPodForDRC(drc, f.Client)
//			Eventually(drc2.IsDRCPodBeDeleted(f, drc), "5m", "10s").ShouldNot(HaveOccurred())
//			Eventually(drc2.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
//			// goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
//			// Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
//		})
//		It("should scale up a DistributedRedisCluster", func() {
//			drc2.ScaleUPDRC(drc)
//			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
//			Eventually(drc2.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
//			// goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
//			// Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
//		})
//		Context("when the scale up succeeded", func() {
//			It("should scale down a DistributedRedisCluster", func() {
//				drc2.ScaleUPDown(drc)
//				Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
//				Eventually(drc2.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
//				// goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
//				// Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
//			})
//		})
//		It("should reset the DistributedRedisCluster password", func() {
//			newPassword := utils.RandString(8)
//			Ω(f.CreateRedisClusterPassword(f.NewPasswordName(), newPassword)).Should(Succeed())
//			drc2.ResetPassword(drc, f.NewPasswordName())
//			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
//			time.Sleep(5 * time.Second)
//			Eventually(drc2.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
//			// goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), newPassword)
//			// Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
//		})
//		It("should update the DistributedRedisCluster minor version", func() {
//			drc2.RollingUpdateDRC(drc)
//			Ω(f.UpdateRedisCluster(drc)).Should(Succeed())
//			Eventually(drc2.IsDistributedRedisClusterProperly(f, drc), "10m", "10s").ShouldNot(HaveOccurred())
//			// goredis = drctest.NewGoRedisClient(drc.Name, f.Namespace(), goredis.Password())
//			// Expect(drctest.IsDBSizeConsistent(dbsize, goredis)).NotTo(HaveOccurred())
//		})
//	})
//})
