package drc

import (
	"context"
	"fmt"

	// rbacv1 "k8s.io/api/rbac/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	redisv1alpha1 "github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
	"github.com/TykTechnologies/redis-cluster-operator/test/utils"
)

type Framework struct {
	KubeConfig *rest.Config
	Client     client.Client
	nameSpace  string
}

// NewFramework create a new Framework with name
func NewFramework(name string) *Framework {
	namespace := fmt.Sprintf("drce2e-%s-%s", name, utils.RandString(8))
	return &Framework{
		nameSpace: namespace,
	}
}

// BeforeEach runs before each test
func (f *Framework) BeforeEach() error {
	// Get a config to talk to the api-server
	cfg, err := config.GetConfig()
	if err != nil {
		f.Failf("get k8s config err: %s", err)
	}
	f.KubeConfig = cfg
	testClient, err := NewClient(cfg)
	if err != nil {
		f.Failf("get k8s config err: %s", err)
	}
	f.Client = testClient
	if err := f.createTestNamespace(); err != nil {
		return err
	}
	//if err := f.createRBAC(); err != nil {
	//	return err
	//}
	return nil
}

// AfterEach runs after each test
func (f *Framework) AfterEach() error {
	f.Logf("clear rbac in namespace")
	//if err := f.deleteRBAC(); err != nil {
	//	return err
	//}

	if err := f.deleteTestNamespace(); err != nil {
		return err
	}
	f.Logf("test namespace deleted")
	return nil
}

// Logf write log to ginkgo output
func (f *Framework) Logf(format string, a ...interface{}) {
	l := fmt.Sprintf(format, a...)
	utils.Logf("namespace: %s %s", f.Namespace(), l)
}

// Failf write log to ginkgo output
func (f *Framework) Failf(format string, a ...interface{}) {
	l := fmt.Sprintf(format, a...)
	utils.Failf("namespace: %s %s", f.Namespace(), l)
}

// Namespace returns the test namespace name
func (f *Framework) Namespace() string {
	return f.nameSpace
}

// CreateRedisCluster creates a DistributedRedisCluster in test namespace
func (f *Framework) CreateRedisCluster(instance *redisv1alpha1.DistributedRedisCluster) error {
	f.Logf("Creating DistributedRedisCluster %s", instance.Name)
	result := &redisv1alpha1.DistributedRedisCluster{}
	err := f.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: f.Namespace(),
		Name:      instance.Name,
	}, result)
	if errors.IsNotFound(err) {
		return f.Client.Create(context.TODO(), instance)
	}
	return err
}

// CreateRedisClusterCleaup creates a RedisClusterCleanup in test namespace
func (f *Framework) CreateRedisClusterCleaup(instance *redisv1alpha1.RedisClusterCleanup) error {
	f.Logf("Creating RedisClusterCleaup %s", instance.Name)
	result := &redisv1alpha1.RedisClusterCleanup{}
	err := f.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: f.Namespace(),
		Name:      instance.Name,
	}, result)
	if errors.IsNotFound(err) {
		return f.Client.Create(context.TODO(), instance)
	}
	return err
}

// CreateRedisClusterPassword creates a password for DistributedRedisCluster
func (f *Framework) CreateRedisClusterPassword(name, password string) error {
	f.Logf("Creating DistributedRedisCluster secret %s", name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace(),
		},
		StringData: map[string]string{
			passwordKey: password,
		},
		Type: "Opaque",
	}
	result := &corev1.Secret{}
	err := f.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: f.Namespace(),
		Name:      name,
	}, result)
	if errors.IsNotFound(err) {
		return f.Client.Create(context.TODO(), secret)
	}
	return err
}

// UpdateRedisCluster update a DistributedRedisCluster in test namespace
func (f *Framework) UpdateRedisCluster(instance *redisv1alpha1.DistributedRedisCluster) error {
	f.Logf("updating DistributedRedisCluster %s", instance.Name)
	cluster := &redisv1alpha1.DistributedRedisCluster{}
	if err := f.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: f.Namespace(),
		Name:      instance.Name,
	}, cluster); err != nil {
		return err
	}
	cluster.Spec = instance.Spec
	return f.Client.Update(context.TODO(), cluster)
}

// DeleteRedisCluster delete a DistributedRedisCluster in test namespace
func (f *Framework) DeleteRedisCluster(instance *redisv1alpha1.DistributedRedisCluster) error {
	f.Logf("deleting DistributedRedisCluster %s", instance.Name)
	return f.Client.Delete(context.TODO(), instance)
}

func (f *Framework) GetDRCPodsByLabels(labels map[string]string) (*corev1.PodList, error) {
	foundPods := &corev1.PodList{}
	err := f.Client.List(context.TODO(), foundPods, client.InNamespace(f.nameSpace), client.MatchingLabels(labels))
	return foundPods, err
}

func (f *Framework) GetDRCStatefulSetByLabels(labels map[string]string) (*appsv1.StatefulSetList, error) {
	foundSts := &appsv1.StatefulSetList{}
	err := f.Client.List(context.TODO(), foundSts, client.InNamespace(f.nameSpace), client.MatchingLabels(labels))
	return foundSts, err
}

func (f *Framework) createTestNamespace() error {
	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: f.Namespace()},
	}
	return f.Client.Create(context.TODO(), nsSpec)
}

func (f *Framework) deleteTestNamespace() error {
	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: f.Namespace()},
	}
	return f.Client.Delete(context.TODO(), nsSpec)
}

func (f *Framework) PasswordName() string {
	return "redis-admin-passwd"
}

func (f *Framework) NewPasswordName() string {
	return "redis-admin-newpasswd"
}

// func (f *Framework) deleteRBAC() error {
// 	rbSpec := &rbacv1.RoleBinding{
// 		ObjectMeta: metav1.ObjectMeta{Name: f.rolebindingName(), Namespace: f.nameSpace},
// 	}
// 	if err := f.Client.Delete(context.TODO(), rbSpec); err != nil {
// 		return err
// 	}
// 	role := &rbacv1.ClusterRole{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      f.roleName(),
// 			Namespace: f.nameSpace,
// 		},
// 	}
// 	return f.Client.Delete(context.TODO(), role)
// }

// // CreateRedisClusterBackup creates a RedisClusterBackup in test namespace
// func (f *Framework) CreateRedisClusterBackup(instance *redisv1alpha1.RedisClusterBackup) error {
// 	f.Logf("Creating RedisClusterBackup %s", instance.Name)
// 	result := &redisv1alpha1.RedisClusterBackup{}
// 	err := f.Client.Get(context.TODO(), types.NamespacedName{
// 		Namespace: f.Namespace(),
// 		Name:      instance.Name,
// 	}, result)
// 	if errors.IsNotFound(err) {
// 		return f.Client.Create(context.TODO(), instance)
// 	}
// 	return err
// }

// // CreateS3Secret creates a secret for RedisClusterBackup
// func (f *Framework) CreateS3Secret(id, key string) error {
// 	name := f.S3SecretName()
// 	f.Logf("Creating RedisClusterBackup secret %s", name)
// 	secret := &corev1.Secret{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: f.Namespace(),
// 		},
// 		StringData: map[string]string{
// 			S3ID:  id,
// 			S3KEY: key,
// 		},
// 		Type: "Opaque",
// 	}
// 	result := &corev1.Secret{}
// 	err := f.Client.Get(context.TODO(), types.NamespacedName{
// 		Namespace: f.Namespace(),
// 		Name:      name,
// 	}, result)
// 	if errors.IsNotFound(err) {
// 		return f.Client.Create(context.TODO(), secret)
// 	}
// 	return err
// }

// func (f *Framework) createRBAC() error {
// 	role := &rbacv1.ClusterRole{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      f.roleName(),
// 			Namespace: f.nameSpace,
// 		},
// 		Rules: []rbacv1.PolicyRule{
// 			{
// 				Verbs:     []string{"get", "list", "watch", "delete", "create", "patch", "update"},
// 				APIGroups: []string{""},
// 				Resources: []string{"pods", "secrets", "endpoints",
// 					"persistentvolumeclaims", "configmaps", "services", "namespaces"},
// 			},
// 			{
// 				Verbs:     []string{"get", "list", "watch", "delete", "create", "patch", "update"},
// 				APIGroups: []string{"batch"},
// 				Resources: []string{"jobs"},
// 			},
// 			{
// 				Verbs:     []string{"get", "list", "watch", "delete", "create", "patch", "update"},
// 				APIGroups: []string{"apps"},
// 				Resources: []string{"deployments", "replicasets", "statefulsets"},
// 			},
// 			{
// 				Verbs:     []string{"get", "list", "watch", "delete", "create", "patch", "update"},
// 				APIGroups: []string{"redis.kun"},
// 				Resources: []string{"*"},
// 			},
// 		},
// 	}
// 	if err := f.Client.Create(context.TODO(), role); err != nil {
// 		return err
// 	}

// 	rbSpec := &rbacv1.RoleBinding{
// 		ObjectMeta: metav1.ObjectMeta{Name: f.rolebindingName(), Namespace: f.nameSpace},
// 		RoleRef: rbacv1.RoleRef{
// 			APIGroup: "rbac.authorization.k8s.io",
// 			Kind:     "ClusterRole",
// 			Name:     f.roleName(),
// 		},
// 		Subjects: []rbacv1.Subject{{
// 			APIGroup:  "rbac.authorization.k8s.io",
// 			Kind:      "Group",
// 			Name:      fmt.Sprintf("system:serviceaccounts:%s", f.nameSpace),
// 			Namespace: f.nameSpace,
// 		}},
// 	}
// 	if err := f.Client.Create(context.TODO(), rbSpec); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (f *Framework) rolebindingName() string {
// 	return fmt.Sprintf("cr-redis~g-%s", f.nameSpace)
// }

// func (f *Framework) roleName() string {
// 	return "cr-redis"
// }
