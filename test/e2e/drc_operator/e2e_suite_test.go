/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
...
*/

package drc_operator_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/TykTechnologies/redis-cluster-operator/test/utils"
)

var (
	namespace    = "redis-cluster-operator-system"
	projectimage = "tykio/redis-cluster-operator:v0.0.0-teste2e"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	fmt.Fprintln(GinkgoWriter, "Starting redis-cluster-operator suite")
	RunSpecs(t, "Operator and DRC E2E Suite")
}

var _ = BeforeSuite(func() {

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	if v, ok := os.LookupEnv("IMG"); ok {
		projectimage = v
	}

	By("creating manager namespace")
	cmd := exec.Command("kubectl", "create", "namespace", namespace)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	By("building the manager (Operator) image")
	cmd = exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	By("loading the manager (Operator) image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectimage)
	Expect(err).NotTo(HaveOccurred())

	By("installing CRDs")
	cmd = exec.Command("make", "install")
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	By("deploying the controller-manager")
	cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	// Optional sleep to let the operator settle
	time.Sleep(5 * time.Second)
})

var _ = AfterSuite(func() {

	By("undeploy the controller-manager")
	cmd := exec.Command("make", "undeploy")
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

})
