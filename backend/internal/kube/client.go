package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"devops-system/backend/internal/models"
)

const requestTimeout = 12 * time.Second
const defaultListLimit int64 = 500
const maxReplicas = int64(1<<31 - 1)

// ClusterCredential is the minimum credential surface the handler passes to client-go.
type ClusterCredential struct {
	APIServer      string
	CredentialType string
	KubeConfig     string
	Token          string
}

type CheckResult struct {
	Status               string                 `json:"status"`
	Version              string                 `json:"version"`
	PermissionSummary    map[string]interface{} `json:"permissionSummary"`
	CertificateExpiresAt *time.Time             `json:"certificateExpiresAt,omitempty"`
	LatencyMS            int64                  `json:"latencyMs"`
}

type ActionRequest struct {
	Namespace string
	Kind      string
	Name      string
	Action    string
	DryRun    bool
	Params    map[string]interface{}
}

type ManifestRequest struct {
	Action    string
	Namespace string
	DryRun    bool
	Manifest  map[string]interface{}
}

type SyncResult struct {
	Snapshots []models.KubernetesResourceSnapshot
	Warnings  []string
}

type Client struct {
	Clientset     kubernetes.Interface
	DynamicClient dynamic.Interface
	RESTMapper    meta.RESTMapper
}

func IsMockAPIServer(apiServer string) bool {
	return strings.HasPrefix(strings.TrimSpace(apiServer), "mock://")
}

func (client Client) Check(ctx context.Context, credential ClusterCredential) (CheckResult, error) {
	startedAt := time.Now()
	clientset, err := client.clientsetFor(credential)
	if err != nil {
		return CheckResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return CheckResult{}, fmt.Errorf("get kubernetes server version failed: %w", err)
	}
	readOK := true
	if _, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1}); err != nil {
		readOK = false
	}
	return CheckResult{
		Status:    "connected",
		Version:   version.GitVersion,
		LatencyMS: time.Since(startedAt).Milliseconds(),
		PermissionSummary: map[string]interface{}{
			"read":  readOK,
			"write": "action-specific",
			"mode":  "client-go",
		},
	}, nil
}

func (client Client) SyncSnapshots(ctx context.Context, credential ClusterCredential, cluster models.KubernetesCluster, now time.Time) (SyncResult, error) {
	clientset, err := client.clientsetFor(credential)
	if err != nil {
		return SyncResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout*2)
	defer cancel()
	snapshots := make([]models.KubernetesResourceSnapshot, 0, 128)
	warnings := make([]string, 0)
	collectors := []func(context.Context, kubernetes.Interface, models.KubernetesCluster, time.Time) ([]models.KubernetesResourceSnapshot, error){
		syncNodes, syncNamespaces, syncDeployments, syncStatefulSets, syncDaemonSets, syncPods, syncServices, syncIngresses, syncPVCs, syncPVs, syncConfigMaps, syncSecrets,
	}
	for _, collect := range collectors {
		items, err := collect(ctx, clientset, cluster, now)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		snapshots = append(snapshots, items...)
	}
	if len(snapshots) == 0 && len(warnings) > 0 {
		return SyncResult{}, fmt.Errorf("all kubernetes resource collectors failed: %s", strings.Join(warnings, "; "))
	}
	return SyncResult{Snapshots: snapshots, Warnings: warnings}, nil
}

func (client Client) ExecuteAction(ctx context.Context, credential ClusterCredential, req ActionRequest) (map[string]interface{}, error) {
	clientset, err := client.clientsetFor(credential)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	dryRun := dryRunOptions(req.DryRun)
	result := map[string]interface{}{"dryRun": req.DryRun, "kind": req.Kind, "name": req.Name, "namespace": req.Namespace}
	switch req.Kind {
	case "Deployment":
		return executeDeploymentAction(ctx, clientset, req, dryRun, result)
	case "StatefulSet":
		return executeStatefulSetAction(ctx, clientset, req, dryRun, result)
	case "DaemonSet":
		return executeDaemonSetAction(ctx, clientset, req, dryRun, result)
	case "Pod":
		return executePodAction(ctx, clientset, req, dryRun, result)
	case "Node":
		return executeNodeAction(ctx, clientset, req, dryRun, result)
	case "Namespace":
		if req.Action != "delete" {
			return nil, fmt.Errorf("unsupported namespace action %s", req.Action)
		}
		return result, clientset.CoreV1().Namespaces().Delete(ctx, req.Name, metav1.DeleteOptions{DryRun: dryRun})
	case "ConfigMap":
		if req.Action != "delete" {
			return nil, fmt.Errorf("unsupported configmap action %s", req.Action)
		}
		return result, clientset.CoreV1().ConfigMaps(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{DryRun: dryRun})
	case "Secret":
		if req.Action != "delete" {
			return nil, fmt.Errorf("unsupported secret action %s", req.Action)
		}
		return result, clientset.CoreV1().Secrets(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{DryRun: dryRun})
	case "PVC":
		if req.Action != "delete" {
			return nil, fmt.Errorf("unsupported pvc action %s", req.Action)
		}
		return result, clientset.CoreV1().PersistentVolumeClaims(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{DryRun: dryRun})
	case "PV":
		if req.Action != "delete" {
			return nil, fmt.Errorf("unsupported pv action %s", req.Action)
		}
		return result, clientset.CoreV1().PersistentVolumes().Delete(ctx, req.Name, metav1.DeleteOptions{DryRun: dryRun})
	default:
		return nil, fmt.Errorf("unsupported kubernetes kind %s", req.Kind)
	}
}

func (client Client) ExecuteManifest(ctx context.Context, credential ClusterCredential, req ManifestRequest) (map[string]interface{}, error) {
	dynamicClient, mapper, err := client.dynamicFor(credential)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	action := strings.ToLower(strings.TrimSpace(req.Action))
	obj, mapping, resourceClient, err := manifestResource(dynamicClient, mapper, req)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"action":     action,
		"dryRun":     req.DryRun,
		"apiVersion": obj.GetAPIVersion(),
		"kind":       obj.GetKind(),
		"namespace":  obj.GetNamespace(),
		"name":       obj.GetName(),
		"resource":   mapping.Resource.Resource,
	}
	switch action {
	case "create":
		created, err := resourceClient.Create(ctx, obj, metav1.CreateOptions{DryRun: dryRunOptions(req.DryRun)})
		if err != nil {
			return nil, err
		}
		result["object"] = sanitizeUnstructuredObject(created)
		result["resourceVersion"] = created.GetResourceVersion()
	case "update":
		if obj.GetResourceVersion() == "" {
			current, err := resourceClient.Get(ctx, obj.GetName(), metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			obj.SetResourceVersion(current.GetResourceVersion())
		}
		updated, err := resourceClient.Update(ctx, obj, metav1.UpdateOptions{DryRun: dryRunOptions(req.DryRun)})
		if err != nil {
			return nil, err
		}
		result["object"] = sanitizeUnstructuredObject(updated)
		result["resourceVersion"] = updated.GetResourceVersion()
	case "delete":
		err := resourceClient.Delete(ctx, obj.GetName(), metav1.DeleteOptions{DryRun: dryRunOptions(req.DryRun)})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported manifest action %s", req.Action)
	}
	return result, nil
}

func (client Client) GetManifest(ctx context.Context, credential ClusterCredential, req ManifestRequest) (map[string]interface{}, error) {
	dynamicClient, mapper, err := client.dynamicFor(credential)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	obj, mapping, resourceClient, err := manifestResource(dynamicClient, mapper, req)
	if err != nil {
		return nil, err
	}
	current, err := resourceClient.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"apiVersion":      current.GetAPIVersion(),
		"kind":            current.GetKind(),
		"namespace":       current.GetNamespace(),
		"name":            current.GetName(),
		"resource":        mapping.Resource.Resource,
		"resourceVersion": current.GetResourceVersion(),
		"object":          sanitizeUnstructuredObject(current),
	}, nil
}

func (client Client) clientsetFor(credential ClusterCredential) (kubernetes.Interface, error) {
	if client.Clientset != nil {
		return client.Clientset, nil
	}
	config, err := restConfigFor(credential)
	if err != nil {
		return nil, err
	}
	config.Timeout = requestTimeout
	return kubernetes.NewForConfig(config)
}

func (client Client) dynamicFor(credential ClusterCredential) (dynamic.Interface, meta.RESTMapper, error) {
	if client.DynamicClient != nil && client.RESTMapper != nil {
		return client.DynamicClient, client.RESTMapper, nil
	}
	config, err := restConfigFor(credential)
	if err != nil {
		return nil, nil, err
	}
	config.Timeout = requestTimeout
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(clientset.Discovery()))
	return dynamicClient, mapper, nil
}

func restConfigFor(credential ClusterCredential) (*rest.Config, error) {
	if strings.EqualFold(strings.TrimSpace(credential.CredentialType), "token") {
		if strings.TrimSpace(credential.APIServer) == "" || strings.TrimSpace(credential.Token) == "" {
			return nil, fmt.Errorf("apiServer and token are required")
		}
		return &rest.Config{Host: strings.TrimSpace(credential.APIServer), BearerToken: strings.TrimSpace(credential.Token)}, nil
	}
	if strings.TrimSpace(credential.KubeConfig) == "" {
		return nil, fmt.Errorf("kubeConfig is required")
	}
	return clientcmd.RESTConfigFromKubeConfig([]byte(credential.KubeConfig))
}

func manifestResource(dynamicClient dynamic.Interface, mapper meta.RESTMapper, req ManifestRequest) (*unstructured.Unstructured, *meta.RESTMapping, dynamic.ResourceInterface, error) {
	if req.Manifest == nil {
		return nil, nil, nil, fmt.Errorf("manifest is required")
	}
	object := runtime.DeepCopyJSON(req.Manifest)
	obj := &unstructured.Unstructured{Object: object}
	if strings.TrimSpace(obj.GetAPIVersion()) == "" || strings.TrimSpace(obj.GetKind()) == "" || strings.TrimSpace(obj.GetName()) == "" {
		return nil, nil, nil, fmt.Errorf("manifest apiVersion, kind and metadata.name are required")
	}
	pruneManifestForWrite(obj, req.Action)
	gv, err := schema.ParseGroupVersion(obj.GetAPIVersion())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid apiVersion: %w", err)
	}
	mapping, err := mapper.RESTMapping(schema.GroupKind{Group: gv.Group, Kind: obj.GetKind()}, gv.Version)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("resolve resource mapping failed: %w", err)
	}
	namespace := strings.TrimSpace(req.Namespace)
	if obj.GetNamespace() != "" {
		namespace = obj.GetNamespace()
	}
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace == "" {
			return nil, nil, nil, fmt.Errorf("namespace is required")
		}
		obj.SetNamespace(namespace)
		return obj, mapping, dynamicClient.Resource(mapping.Resource).Namespace(namespace), nil
	}
	obj.SetNamespace("")
	return obj, mapping, dynamicClient.Resource(mapping.Resource), nil
}

func syncNodes(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.CoreV1().Nodes().List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list nodes failed: %w", err)
		}
		for _, item := range list.Items {
			items = append(items, snapshot(cluster, "", "Node", item.Name, string(item.UID), nodeStatus(item), fmt.Sprintf("cpu=%s memory=%s pods=%s", q(item.Status.Capacity.Cpu()), q(item.Status.Capacity.Memory()), q(item.Status.Capacity.Pods())), item.ResourceVersion, item.Labels, map[string]interface{}{"conditions": nodeConditions(item)}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncNamespaces(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.CoreV1().Namespaces().List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list namespaces failed: %w", err)
		}
		for _, item := range list.Items {
			items = append(items, snapshot(cluster, "", "Namespace", item.Name, string(item.UID), string(item.Status.Phase), "namespace", item.ResourceVersion, item.Labels, map[string]interface{}{"annotations": item.Annotations}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncDeployments(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.AppsV1().Deployments(metav1.NamespaceAll).List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list deployments failed: %w", err)
		}
		for _, item := range list.Items {
			replicas := replicasOrZero(item.Spec.Replicas)
			items = append(items, snapshot(cluster, item.Namespace, "Deployment", item.Name, string(item.UID), workloadStatus(item.Status.AvailableReplicas, replicas), fmt.Sprintf("replicas=%d/%d image=%s", item.Status.AvailableReplicas, replicas, firstContainerImage(item.Spec.Template.Spec.Containers)), item.ResourceVersion, item.Labels, map[string]interface{}{"replicas": replicas, "availableReplicas": item.Status.AvailableReplicas}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncStatefulSets(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.AppsV1().StatefulSets(metav1.NamespaceAll).List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list statefulsets failed: %w", err)
		}
		for _, item := range list.Items {
			replicas := replicasOrZero(item.Spec.Replicas)
			items = append(items, snapshot(cluster, item.Namespace, "StatefulSet", item.Name, string(item.UID), workloadStatus(item.Status.ReadyReplicas, replicas), fmt.Sprintf("ready=%d/%d image=%s", item.Status.ReadyReplicas, replicas, firstContainerImage(item.Spec.Template.Spec.Containers)), item.ResourceVersion, item.Labels, map[string]interface{}{"replicas": replicas, "readyReplicas": item.Status.ReadyReplicas}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncDaemonSets(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.AppsV1().DaemonSets(metav1.NamespaceAll).List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list daemonsets failed: %w", err)
		}
		for _, item := range list.Items {
			items = append(items, snapshot(cluster, item.Namespace, "DaemonSet", item.Name, string(item.UID), workloadStatus(item.Status.NumberReady, item.Status.DesiredNumberScheduled), fmt.Sprintf("ready=%d/%d image=%s", item.Status.NumberReady, item.Status.DesiredNumberScheduled, firstContainerImage(item.Spec.Template.Spec.Containers)), item.ResourceVersion, item.Labels, map[string]interface{}{"desired": item.Status.DesiredNumberScheduled, "ready": item.Status.NumberReady}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncPods(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.CoreV1().Pods(metav1.NamespaceAll).List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list pods failed: %w", err)
		}
		for _, item := range list.Items {
			restarts := podRestartCount(item)
			items = append(items, snapshot(cluster, item.Namespace, "Pod", item.Name, string(item.UID), string(item.Status.Phase), fmt.Sprintf("node=%s restarts=%d image=%s", item.Spec.NodeName, restarts, firstContainerImage(item.Spec.Containers)), item.ResourceVersion, item.Labels, map[string]interface{}{"node": item.Spec.NodeName, "podIP": item.Status.PodIP, "restarts": restarts}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncServices(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.CoreV1().Services(metav1.NamespaceAll).List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list services failed: %w", err)
		}
		for _, item := range list.Items {
			items = append(items, snapshot(cluster, item.Namespace, "Service", item.Name, string(item.UID), "Active", fmt.Sprintf("type=%s ports=%d clusterIP=%s", item.Spec.Type, len(item.Spec.Ports), item.Spec.ClusterIP), item.ResourceVersion, item.Labels, map[string]interface{}{"type": string(item.Spec.Type), "clusterIP": item.Spec.ClusterIP}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncIngresses(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.NetworkingV1().Ingresses(metav1.NamespaceAll).List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list ingresses failed: %w", err)
		}
		for _, item := range list.Items {
			hosts := ingressHosts(item.Spec.Rules)
			items = append(items, snapshot(cluster, item.Namespace, "Ingress", item.Name, string(item.UID), "Ready", fmt.Sprintf("hosts=%s tls=%t", hosts, len(item.Spec.TLS) > 0), item.ResourceVersion, item.Labels, map[string]interface{}{"hosts": hosts, "tls": len(item.Spec.TLS) > 0}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncPVCs(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list pvcs failed: %w", err)
		}
		for _, item := range list.Items {
			storage := ""
			if value, ok := item.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
				storage = value.String()
			}
			items = append(items, snapshot(cluster, item.Namespace, "PVC", item.Name, string(item.UID), string(item.Status.Phase), fmt.Sprintf("storage=%s volume=%s", storage, item.Spec.VolumeName), item.ResourceVersion, item.Labels, map[string]interface{}{"storage": storage, "volume": item.Spec.VolumeName}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncPVs(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.CoreV1().PersistentVolumes().List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list pvs failed: %w", err)
		}
		for _, item := range list.Items {
			storage := ""
			if value, ok := item.Spec.Capacity[corev1.ResourceStorage]; ok {
				storage = value.String()
			}
			claim := ""
			if item.Spec.ClaimRef != nil {
				claim = item.Spec.ClaimRef.Namespace + "/" + item.Spec.ClaimRef.Name
			}
			storageClass := ""
			if item.Spec.StorageClassName != "" {
				storageClass = item.Spec.StorageClassName
			}
			items = append(items, snapshot(cluster, "", "PV", item.Name, string(item.UID), string(item.Status.Phase), fmt.Sprintf("storage=%s claim=%s storageClass=%s", storage, claim, storageClass), item.ResourceVersion, item.Labels, map[string]interface{}{"storage": storage, "claim": claim, "storageClass": storageClass}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncConfigMaps(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.CoreV1().ConfigMaps(metav1.NamespaceAll).List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list configmaps failed: %w", err)
		}
		for _, item := range list.Items {
			items = append(items, snapshot(cluster, item.Namespace, "ConfigMap", item.Name, string(item.UID), "Active", fmt.Sprintf("keys=%d", len(item.Data)+len(item.BinaryData)), item.ResourceVersion, item.Labels, map[string]interface{}{"keys": mapKeys(item.Data), "binaryKeys": mapKeys(item.BinaryData)}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func syncSecrets(ctx context.Context, c kubernetes.Interface, cluster models.KubernetesCluster, now time.Time) ([]models.KubernetesResourceSnapshot, error) {
	items := make([]models.KubernetesResourceSnapshot, 0)
	continueToken := ""
	for {
		list, err := c.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, pagedListOptions(continueToken))
		if err != nil {
			return nil, fmt.Errorf("list secrets failed: %w", err)
		}
		for _, item := range list.Items {
			items = append(items, snapshot(cluster, item.Namespace, "Secret", item.Name, string(item.UID), "Active", fmt.Sprintf("type=%s keys=%d", item.Type, len(item.Data)), item.ResourceVersion, item.Labels, map[string]interface{}{"type": string(item.Type), "keys": mapKeys(item.Data)}, now))
		}
		if list.Continue == "" {
			break
		}
		continueToken = list.Continue
	}
	return items, nil
}

func executeDeploymentAction(ctx context.Context, c kubernetes.Interface, req ActionRequest, dryRun []string, result map[string]interface{}) (map[string]interface{}, error) {
	switch req.Action {
	case "restart":
		patch := restartPatch()
		_, err := c.AppsV1().Deployments(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{DryRun: dryRun})
		return result, err
	case "scale":
		replicas, err := replicasParam(req.Params)
		if err != nil {
			return nil, err
		}
		scale, err := c.AppsV1().Deployments(req.Namespace).GetScale(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		scale.Spec.Replicas = replicas
		_, err = c.AppsV1().Deployments(req.Namespace).UpdateScale(ctx, req.Name, scale, metav1.UpdateOptions{DryRun: dryRun})
		result["replicas"] = replicas
		return result, err
	case "pause", "resume":
		patch := fmt.Sprintf(`{"spec":{"paused":%t}}`, req.Action == "pause")
		_, err := c.AppsV1().Deployments(req.Namespace).Patch(ctx, req.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{DryRun: dryRun})
		return result, err
	case "delete":
		return result, c.AppsV1().Deployments(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{DryRun: dryRun})
	default:
		return nil, fmt.Errorf("unsupported deployment action %s", req.Action)
	}
}

func executeStatefulSetAction(ctx context.Context, c kubernetes.Interface, req ActionRequest, dryRun []string, result map[string]interface{}) (map[string]interface{}, error) {
	switch req.Action {
	case "restart":
		_, err := c.AppsV1().StatefulSets(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, restartPatch(), metav1.PatchOptions{DryRun: dryRun})
		return result, err
	case "scale":
		replicas, err := replicasParam(req.Params)
		if err != nil {
			return nil, err
		}
		scale, err := c.AppsV1().StatefulSets(req.Namespace).GetScale(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		scale.Spec.Replicas = replicas
		_, err = c.AppsV1().StatefulSets(req.Namespace).UpdateScale(ctx, req.Name, scale, metav1.UpdateOptions{DryRun: dryRun})
		result["replicas"] = replicas
		return result, err
	case "delete":
		return result, c.AppsV1().StatefulSets(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{DryRun: dryRun})
	default:
		return nil, fmt.Errorf("unsupported statefulset action %s", req.Action)
	}
}

func executeDaemonSetAction(ctx context.Context, c kubernetes.Interface, req ActionRequest, dryRun []string, result map[string]interface{}) (map[string]interface{}, error) {
	if req.Action != "restart" {
		return nil, fmt.Errorf("unsupported daemonset action %s", req.Action)
	}
	_, err := c.AppsV1().DaemonSets(req.Namespace).Patch(ctx, req.Name, types.StrategicMergePatchType, restartPatch(), metav1.PatchOptions{DryRun: dryRun})
	return result, err
}

func executePodAction(ctx context.Context, c kubernetes.Interface, req ActionRequest, dryRun []string, result map[string]interface{}) (map[string]interface{}, error) {
	switch req.Action {
	case "delete":
		return result, c.CoreV1().Pods(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{DryRun: dryRun})
	case "evict":
		return result, c.CoreV1().Pods(req.Namespace).EvictV1(ctx, &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}, DeleteOptions: &metav1.DeleteOptions{DryRun: dryRun}})
	default:
		return nil, fmt.Errorf("unsupported pod action %s", req.Action)
	}
}

func executeNodeAction(ctx context.Context, c kubernetes.Interface, req ActionRequest, dryRun []string, result map[string]interface{}) (map[string]interface{}, error) {
	switch req.Action {
	case "cordon", "uncordon":
		unschedulable := req.Action == "cordon"
		patch := fmt.Sprintf(`{"spec":{"unschedulable":%t}}`, unschedulable)
		_, err := c.CoreV1().Nodes().Patch(ctx, req.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{DryRun: dryRun})
		result["unschedulable"] = unschedulable
		return result, err
	default:
		return nil, fmt.Errorf("unsupported node action %s", req.Action)
	}
}

func dryRunOptions(enabled bool) []string {
	if enabled {
		return []string{metav1.DryRunAll}
	}
	return nil
}

func pagedListOptions(continueToken string) metav1.ListOptions {
	return metav1.ListOptions{Limit: defaultListLimit, Continue: continueToken}
}

func replicasOrZero(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}
func q(value *resource.Quantity) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func snapshot(cluster models.KubernetesCluster, namespace, kind, name, uid, status, spec, version string, labels map[string]string, metadata map[string]interface{}, now time.Time) models.KubernetesResourceSnapshot {
	return models.KubernetesResourceSnapshot{ClusterID: cluster.ID, Namespace: namespace, Kind: kind, Name: name, UID: uid, Status: status, SpecSummary: spec, ResourceVersion: version, Labels: stringMap(labels), Metadata: datatypes.JSONMap(metadata), LastSyncedAt: now}
}

func sanitizeUnstructuredObject(obj *unstructured.Unstructured) map[string]interface{} {
	if obj == nil {
		return map[string]interface{}{}
	}
	out := runtime.DeepCopyJSON(obj.Object)
	removeNestedFields(out, []string{"metadata", "managedFields"}, []string{"status"})
	if strings.EqualFold(obj.GetKind(), "Secret") {
		delete(out, "data")
		delete(out, "stringData")
		out["dataRedacted"] = true
	}
	return out
}

func pruneManifestForWrite(obj *unstructured.Unstructured, action string) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "create", "update":
	default:
		return
	}
	removeNestedFields(obj.Object,
		[]string{"metadata", "managedFields"},
		[]string{"metadata", "creationTimestamp"},
		[]string{"metadata", "generation"},
		[]string{"metadata", "selfLink"},
		[]string{"metadata", "uid"},
		[]string{"status"},
	)
	if strings.EqualFold(action, "create") {
		removeNestedFields(obj.Object, []string{"metadata", "resourceVersion"})
	}
}

func removeNestedFields(object map[string]interface{}, paths ...[]string) {
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		unstructured.RemoveNestedField(object, path...)
	}
}

func stringMap(values map[string]string) datatypes.JSONMap {
	out := datatypes.JSONMap{}
	for key, value := range values {
		out[key] = value
	}
	return out
}
func nodeStatus(node corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return "Ready"
		}
	}
	return "NotReady"
}
func nodeConditions(node corev1.Node) []map[string]string {
	out := []map[string]string{}
	for _, item := range node.Status.Conditions {
		out = append(out, map[string]string{"type": string(item.Type), "status": string(item.Status), "reason": item.Reason})
	}
	return out
}
func workloadStatus(ready int32, desired int32) string {
	if desired == 0 {
		return "ScaledToZero"
	}
	if ready >= desired {
		return "Available"
	}
	return "Progressing"
}
func firstContainerImage(containers []corev1.Container) string {
	if len(containers) == 0 {
		return ""
	}
	return containers[0].Image
}
func podRestartCount(pod corev1.Pod) int32 {
	var total int32
	for _, item := range pod.Status.ContainerStatuses {
		total += item.RestartCount
	}
	return total
}
func ingressHosts(rules []networkingv1.IngressRule) string {
	hosts := []string{}
	for _, rule := range rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	return strings.Join(hosts, ",")
}

func mapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func restartPatch() []byte {
	payload := map[string]interface{}{"spec": map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]string{"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339)}}}}}
	raw, _ := json.Marshal(payload)
	return raw
}

func replicasParam(params map[string]interface{}) (int32, error) {
	value, ok := params["replicas"]
	if !ok {
		return 0, fmt.Errorf("params.replicas is required")
	}
	switch typed := value.(type) {
	case int:
		if typed < 0 || int64(typed) > maxReplicas {
			return 0, fmt.Errorf("params.replicas must be non-negative integer")
		}
		return int32(typed), nil
	case int32:
		if typed < 0 {
			return 0, fmt.Errorf("params.replicas must be non-negative integer")
		}
		return typed, nil
	case int64:
		if typed < 0 || typed > maxReplicas {
			return 0, fmt.Errorf("params.replicas must be non-negative integer")
		}
		return int32(typed), nil
	case float64:
		if typed < 0 || typed > float64(maxReplicas) || typed != float64(int32(typed)) {
			return 0, fmt.Errorf("params.replicas must be non-negative integer")
		}
		return int32(typed), nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 32)
		if err != nil || parsed < 0 {
			return 0, fmt.Errorf("params.replicas must be non-negative integer")
		}
		return int32(parsed), nil
	default:
		return 0, fmt.Errorf("params.replicas must be non-negative integer")
	}
}
