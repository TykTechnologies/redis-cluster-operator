package utils

const (
	// AnnotationScope annotation name for defining instance scope. Used for specifying cluster wide clusters.
	// A namespace-scoped operator watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide.
	AnnotationScope = "redis.kun/scope"
	//AnnotationClusterScoped annotation value for cluster wide clusters.
	AnnotationClusterScoped = "cluster-scoped"
)

var isClusterScoped = true

func IsClusterScoped() bool {
	return isClusterScoped
}

func SetClusterScoped(namespace string) {
	if namespace != "" {
		isClusterScoped = false
	}
}

func ShoudManage(annotations map[string]string) bool {
	if v, ok := annotations[AnnotationScope]; ok {
		if IsClusterScoped() {
			return v == AnnotationClusterScoped
		}
	} else {
		if !IsClusterScoped() {
			return true
		}
	}
	return false
}
