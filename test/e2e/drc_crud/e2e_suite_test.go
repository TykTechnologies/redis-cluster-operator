package drc_crud_test

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	redisv1alpha1 "github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
	drctest "github.com/TykTechnologies/redis-cluster-operator/test/e2e/drc_crud/drc"
)

var f *drctest.Framework
var drc *redisv1alpha1.DistributedRedisCluster

func TestDrc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Drc Suite")
}

var _ = BeforeSuite(func() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	f = drctest.NewFramework("test")
	if err := f.BeforeEach(); err != nil {
		f.Failf("Framework BeforeEach err: %s", err.Error())
	}
})

var _ = AfterSuite(func() {
	if err := f.DeleteRedisCluster(drc); err != nil {
		f.Logf("deleting DistributedRedisCluster err: %s", err.Error())
	}
	if err := f.AfterEach(); err != nil {
		f.Failf("Framework AfterSuite err: %s", err.Error())
	}
})
