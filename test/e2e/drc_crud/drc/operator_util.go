package drc

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	redisv1alpha1 "github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
	"github.com/TykTechnologies/redis-cluster-operator/internal/config"
	"github.com/TykTechnologies/redis-cluster-operator/internal/redisutil"
	"github.com/TykTechnologies/redis-cluster-operator/internal/utils"
	testutils "github.com/TykTechnologies/redis-cluster-operator/test/utils"
)

const (
	Redis5_0_4  = "uhub.service.ucloud.cn/operator/redis:5.0.4-alpine"
	Redis5_0_6  = "uhub.service.ucloud.cn/operator/redis:5.0.6-alpine"
	Redis6_2_6  = "redis:6.2.6"
	Redis6_2_12 = "redis:6.2.12"

	passwordKey = "password"

	// exporterImage = "uhub.service.ucloud.cn/operator/redis_exporter:latest"
	// BackupImage = "uhub.service.ucloud.cn/operator/redis-tools:5.0.4"
	// S3ID        = "AWS_ACCESS_KEY_ID"
	// S3KEY       = "AWS_SECRET_ACCESS_KEY"
	// S3ENDPOINT  = "S3_ENDPOINT"
	// S3BUCKET    = "S3_BUCKET"

	// RedisRenameCommandsDefaultPath default path to volume storing rename commands
	RedisRenameCommandsDefaultPath = "test/e2e/drc_crud/drc"
	// RedisRenameCommandsDefaultFile default file name containing rename commands
	RedisRenameCommandsDefaultFile = "rename.conf"
)

var (
	renameCommandsPath string
	renameCommandsFile string
)

func init() {
	flag.StringVar(&renameCommandsPath, "rename-command-path", RedisRenameCommandsDefaultPath, "Path to the folder where rename-commands option for redis are available")
	flag.StringVar(&renameCommandsFile, "rename-command-file", RedisRenameCommandsDefaultFile, "Name of the file where rename-commands option for redis are available, disabled if empty")
}

var logger = logf.Log.WithName("e2e-test")

func NewDistributedRedisCluster(name, namespace, image, passwordName string, masterSize, clusterReplicas int32) *redisv1alpha1.DistributedRedisCluster {
	configParams := map[string]string{
		"hz":         "11",
		"maxclients": "101",
	}
	storageClassName := "standard"
	return &redisv1alpha1.DistributedRedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"redis.kun/scope": "cluster-scoped",
			},
		},
		Spec: redisv1alpha1.DistributedRedisClusterSpec{
			Image:           image,
			MasterSize:      masterSize,
			ClusterReplicas: clusterReplicas,
			Config:          configParams,
			PasswordSecret:  &corev1.LocalObjectReference{Name: passwordName},
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("300Mi"),
				},
			},
			Storage: &redisv1alpha1.RedisStorage{
				Type:        "persistent-claim",
				Size:        resource.MustParse("2Gi"),
				Class:       storageClassName,
				DeleteClaim: true,
			},
			// Monitor: &redisv1alpha1.AgentSpec{
			// 	Image: exporterImage,
			// 	Prometheus: &redisv1alpha1.PrometheusSpec{
			// 		Port: 9121,
			// 	},
			// },
			// Annotations: map[string]string{
			// 	"prometheus.io/app-metrics":      "true",
			// 	"prometheus.io/app-metrics-path": "/metrics",
			// 	"prometheus.io/app-metrics-port": "9121",
			// 	"prometheus.io/scrape":           "true",
			// },
		},
	}
}

func IsDistributedRedisClusterProperly(f *Framework, drc *redisv1alpha1.DistributedRedisCluster) func() error {
	return func() error {
		result := &redisv1alpha1.DistributedRedisCluster{}
		if err := f.Client.Get(context.TODO(), types.NamespacedName{
			Namespace: f.Namespace(),
			Name:      drc.Name,
		}, result); err != nil {
			f.Logf("can not get DistributedRedisCluster err: %s", err.Error())
			return err
		}
		if result.Status.Status != redisv1alpha1.ClusterStatusOK {
			if result.Status.Status == redisv1alpha1.ClusterStatusKO {
				f.Logf("DistributedRedisCluster %s is %s, reason: %s", drc.Name, result.Status.Status, result.Status.Reason)
			}
			return testutils.LogAndReturnErrorf("DistributedRedisCluster %s status not healthy, current: %s", drc.Name, result.Status.Status)
		}
		stsList, err := f.GetDRCStatefulSetByLabels(getLabels(drc))
		if err != nil {
			f.Logf("GetDRCStatefulSetByLabels err: %s", err)
			return err
		}
		for _, sts := range stsList.Items {
			if sts.Status.ReadyReplicas != (drc.Spec.ClusterReplicas + 1) {
				return testutils.LogAndReturnErrorf("DistributedRedisCluster %s wrong ready replicas, want: %d, got: %d",
					drc.Name, drc.Spec.ClusterReplicas+1, sts.Status.ReadyReplicas)
			}
			if sts.Status.CurrentReplicas != (drc.Spec.ClusterReplicas + 1) {
				return testutils.LogAndReturnErrorf("DistributedRedisCluster %s wrong current replicas, want: %d, got: %d",
					drc.Name, drc.Spec.ClusterReplicas+1, sts.Status.ReadyReplicas)
			}
		}

		password, err := getClusterPassword(f.Client, drc)
		if err != nil {
			f.Logf("getClusterPassword err: %s", err)
			return err
		}
		podList, err := f.GetDRCPodsByLabels(getLabels(drc))
		if err != nil {
			f.Logf("GetDRCPodsByLabels err: %s", err)
			return err
		}
		if len(podList.Items) != int(drc.Spec.MasterSize*(drc.Spec.ClusterReplicas+1)) {
			return testutils.LogAndReturnErrorf("DistributedRedisCluster %s wrong node number, masterSize %d, clusterReplicas %d, got node number %d",
				drc.Name, drc.Spec.MasterSize, drc.Spec.ClusterReplicas, len(podList.Items))
		}
		redisconf := &config.Redis{
			DialTimeout:        5000,
			RenameCommandsFile: renameCommandsFile,
			RenameCommandsPath: renameCommandsPath,
		}
		redisAdmin, err := NewRedisAdmin(podList.Items, password, redisconf, logger)
		if err != nil {
			f.Logf("NewRedisAdmin err: %s", err)
			return err
		}
		if _, err := redisAdmin.GetClusterInfos(); err != nil {
			f.Logf("DistributedRedisCluster Cluster nodes: %s", err)
			return err
		}
		for addr, c := range redisAdmin.Connections().GetAll() {
			configs, err := redisAdmin.GetAllConfig(c, addr)
			if err != nil {
				f.Logf("DistributedRedisCluster CONFIG GET: %s", err)
				return err
			}
			for key, value := range drc.Spec.Config {
				if value != configs[key] {
					return testutils.LogAndReturnErrorf("DistributedRedisCluster %s wrong redis config, key: %s, want: %s, got: %s", drc.Name, key, value, configs[key])
				}
			}
		}

		drc.Spec = result.Spec
		return nil
	}
}

func getLabels(cluster *redisv1alpha1.DistributedRedisCluster) map[string]string {
	dynLabels := map[string]string{
		redisv1alpha1.LabelClusterName:  cluster.Name,
		redisv1alpha1.LabelManagedByKey: redisv1alpha1.OperatorName,
	}
	return utils.MergeLabels(dynLabels, cluster.Labels)
}

// NewRedisAdmin builds and returns new redis.Admin from the list of pods
func NewRedisAdmin(pods []corev1.Pod, password string, cfg *config.Redis, reqLogger logr.Logger) (redisutil.IAdmin, error) {
	nodesAddrs := []string{}
	for _, pod := range pods {
		redisPort := redisutil.DefaultRedisPort
		for _, container := range pod.Spec.Containers {
			if container.Name == "redis" {
				for _, port := range container.Ports {
					if port.Name == "client" {
						redisPort = fmt.Sprintf("%d", port.ContainerPort)
					}
				}
			}
		}
		reqLogger.V(4).Info("append redis admin addr", "addr", pod.Status.PodIP, "port", redisPort)
		nodesAddrs = append(nodesAddrs, net.JoinHostPort(pod.Status.PodIP, redisPort))
	}
	adminConfig := redisutil.AdminOptions{
		ConnectionTimeout:  time.Duration(cfg.DialTimeout) * time.Millisecond,
		RenameCommandsFile: cfg.GetRenameCommandsFile(),
		Password:           password,
	}

	return redisutil.NewAdmin(nodesAddrs, &adminConfig, reqLogger), nil
}

func getClusterPassword(client client.Client, cluster *redisv1alpha1.DistributedRedisCluster) (string, error) {
	if cluster.Spec.PasswordSecret == nil {
		return "", nil
	}
	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      cluster.Spec.PasswordSecret.Name,
		Namespace: cluster.Namespace,
	}, secret)
	if err != nil {
		return "", err
	}
	return string(secret.Data[passwordKey]), nil
}

func ChangeDRCRedisConfig(drc *redisv1alpha1.DistributedRedisCluster) {
	drc.Spec.Config["hz"] = "15"
	drc.Spec.Config["maxclients"] = "105"
}

func ScaleUPDRC(drc *redisv1alpha1.DistributedRedisCluster) {
	drc.Spec.MasterSize = 4
}

func ScaleUPDown(drc *redisv1alpha1.DistributedRedisCluster) {
	drc.Spec.MasterSize = 3
}

func ResetPassword(drc *redisv1alpha1.DistributedRedisCluster, passwordSecret string) {
	drc.Spec.PasswordSecret = &corev1.LocalObjectReference{Name: passwordSecret}
}

func RollingUpdateDRC(drc *redisv1alpha1.DistributedRedisCluster) {
	drc.Spec.Image = Redis6_2_12
}

func DeleteMasterPodForDRC(drc *redisv1alpha1.DistributedRedisCluster, client client.Client) {
	result := &redisv1alpha1.DistributedRedisCluster{}
	if err := client.Get(context.TODO(), types.NamespacedName{
		Namespace: drc.Namespace,
		Name:      drc.Name,
	}, result); err != nil {
		testutils.Failf("can not get DistributedRedisCluster err: %s", err.Error())
	}
	for _, node := range result.Status.Nodes {
		if node.Role == redisv1alpha1.RedisClusterNodeRoleMaster {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      node.PodName,
					Namespace: drc.Namespace,
				},
			}
			testutils.Logf("deleting pod %s", node.PodName)
			if err := client.Delete(context.TODO(), pod); err != nil {
				testutils.Failf("can not delete DistributedRedisCluster's pod, err: %s", err)
			}
		}
	}
}

func IsDRCPodBeDeleted(f *Framework, drc *redisv1alpha1.DistributedRedisCluster) func() error {
	return func() error {
		stsList, err := f.GetDRCStatefulSetByLabels(getLabels(drc))
		if err != nil {
			return testutils.LogAndReturnErrorf("GetDRCStatefulSetByLabels err: %s", err)
		}
		for _, sts := range stsList.Items {
			if sts.Status.ReadyReplicas != (drc.Spec.ClusterReplicas + 1) {
				return nil
			}
		}
		return testutils.LogAndReturnErrorf("StatefulSet's Pod still running")
	}
}

func NewGoRedisClient(svc, namespaces, password string) *GoRedis {
	addr := fmt.Sprintf("%s.%s.svc.%s:6379", svc, namespaces, "cluster.local")
	return NewGoRedis(addr, password)
}

func IsDBSizeConsistent(originalDBSize int64, goredis *GoRedis) error {
	curDBSize, err := goredis.DBSize()
	if err != nil {
		return err
	}
	if curDBSize != originalDBSize {
		return testutils.LogAndReturnErrorf("DBSize do not Equal current: %d, original: %d", curDBSize, originalDBSize)
	}
	return nil
}

// func RestoreDRC(drc *redisv1alpha1.DistributedRedisCluster, drcb *redisv1alpha1.RedisClusterBackup) *redisv1alpha1.DistributedRedisCluster {
// 	name := RandString(8)
// 	return &redisv1alpha1.DistributedRedisCluster{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: drc.Namespace,
// 			Annotations: map[string]string{
// 				"redis.kun/scope": "cluster-scoped",
// 			},
// 		},
// 		Spec: redisv1alpha1.DistributedRedisClusterSpec{
// 			Image:           drc.Spec.Image,
// 			MasterSize:      drc.Spec.MasterSize,
// 			ClusterReplicas: drc.Spec.ClusterReplicas,
// 			Config:          drc.Spec.Config,
// 			PasswordSecret:  drc.Spec.PasswordSecret,
// 			Resources:       drc.Spec.Resources,
// 			Storage:         drc.Spec.Storage,
// 			Monitor:         drc.Spec.Monitor,
// 			Annotations:     drc.Spec.Annotations,
// 			Init: &redisv1alpha1.InitSpec{BackupSource: &redisv1alpha1.BackupSourceSpec{
// 				Namespace: drcb.Namespace,
// 				Name:      drcb.Name,
// 			}},
// 		},
// 	}
// }

// func NewRedisClusterBackup(name, namespace, image, drcName, storageSecretName, s3Endpoint, s3Bucket string) *redisv1alpha1.RedisClusterBackup {
// 	return &redisv1alpha1.RedisClusterBackup{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: namespace,
// 			Annotations: map[string]string{
// 				"redis.kun/scope": "cluster-scoped",
// 			},
// 		},
// 		Spec: redisv1alpha1.RedisClusterBackupSpec{
// 			Image:            image,
// 			RedisClusterName: drcName,
// 			Backend: store.Backend{
// 				StorageSecretName: storageSecretName,
// 				S3: &store.S3Spec{
// 					Endpoint: s3Endpoint,
// 					Bucket:   s3Bucket,
// 				},
// 			},
// 		},
// 	}

// }

// func IsRedisClusterBackupProperly(f *Framework, drcb *redisv1alpha1.RedisClusterBackup) func() error {
// 	return func() error {
// 		result := &redisv1alpha1.RedisClusterBackup{}
// 		if err := f.Client.Get(context.TODO(), types.NamespacedName{
// 			Namespace: f.Namespace(),
// 			Name:      drcb.Name,
// 		}, result); err != nil {
// 			f.Logf("can not get DistributedRedisCluster err: %s", err.Error())
// 			return err
// 		}
// 		if result.Status.Phase != redisv1alpha1.BackupPhaseSucceeded {
// 			return LogAndReturnErrorf("RedisClusterBackup %s status not Succeeded, current: %s", drcb.Name, result.Status.Phase)
// 		}
// 		return nil
// 	}
// }
