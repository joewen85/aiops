package kube

import (
	"context"
	"errors"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"devops-system/backend/internal/models"
)

func TestClientSyncSnapshotsWithFakeClient(t *testing.T) {
	replicas := int32(2)
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default", UID: "ns-1"}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1", UID: "node-1"}, Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default", UID: "deploy-1"}, Spec: appsv1.DeploymentSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "nginx:latest"}}}}}, Status: appsv1.DeploymentStatus{AvailableReplicas: 2}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "default", UID: "secret-1"}, Data: map[string][]byte{"password": []byte("should-not-be-returned")}},
		&corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "pv-data", UID: "pv-1"}, Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound}},
	)

	result, err := (Client{Clientset: clientset}).SyncSnapshots(context.Background(), ClusterCredential{}, models.KubernetesCluster{BaseModel: models.BaseModel{ID: 10}, Env: "dev"}, time.Now())
	if err != nil {
		t.Fatalf("sync snapshots failed: %v", err)
	}
	snapshots := result.Snapshots
	if len(snapshots) < 4 {
		t.Fatalf("expected snapshots from fake client, got=%d", len(snapshots))
	}
	foundPV := false
	for _, item := range snapshots {
		if item.Kind == "Secret" && item.Metadata["password"] != nil {
			t.Fatalf("secret snapshot must not expose secret values: %+v", item.Metadata)
		}
		if item.Kind == "PV" && item.Name == "pv-data" {
			foundPV = true
		}
	}
	if !foundPV {
		t.Fatalf("expected pv snapshot to be collected")
	}
}

func TestClientSyncSnapshotsReturnsWarningsForPartialCollectorFailure(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default", UID: "ns-1"}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}},
	)
	clientset.PrependReactor("list", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("rbac denied configmaps")
	})

	result, err := (Client{Clientset: clientset}).SyncSnapshots(context.Background(), ClusterCredential{}, models.KubernetesCluster{BaseModel: models.BaseModel{ID: 10}, Env: "dev"}, time.Now())
	if err != nil {
		t.Fatalf("partial collector failure should not fail whole sync: %v", err)
	}
	if len(result.Snapshots) == 0 {
		t.Fatalf("expected snapshots from successful collectors")
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected one warning, got=%v", result.Warnings)
	}
}

func TestClientExecuteManifestWithDynamicFakeClient(t *testing.T) {
	scheme := runtime.NewScheme()
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{{Group: "platform.aiops.local", Version: "v1"}})
	widgetGVK := schema.GroupVersionKind{Group: "platform.aiops.local", Version: "v1", Kind: "Widget"}
	mapper.Add(widgetGVK, meta.RESTScopeNamespace)
	client := Client{DynamicClient: dynamicfake.NewSimpleDynamicClient(scheme), RESTMapper: mapper}
	manifest := map[string]interface{}{
		"apiVersion": "platform.aiops.local/v1",
		"kind":       "Widget",
		"metadata": map[string]interface{}{
			"name":      "sample",
			"namespace": "default",
		},
		"spec": map[string]interface{}{"replicas": int64(1)},
	}

	result, err := client.ExecuteManifest(context.Background(), ClusterCredential{}, ManifestRequest{Action: "create", Namespace: "default", Manifest: manifest})
	if err != nil {
		t.Fatalf("create manifest failed: %v", err)
	}
	if result["kind"] != "Widget" || result["namespace"] != "default" {
		t.Fatalf("unexpected create result: %+v", result)
	}

	current, err := client.GetManifest(context.Background(), ClusterCredential{}, ManifestRequest{Namespace: "default", Manifest: manifest})
	if err != nil {
		t.Fatalf("get manifest failed: %v", err)
	}
	if current["name"] != "sample" {
		t.Fatalf("unexpected get result: %+v", current)
	}

	manifest["spec"] = map[string]interface{}{"replicas": int64(2)}
	if _, err := client.ExecuteManifest(context.Background(), ClusterCredential{}, ManifestRequest{Action: "update", Namespace: "default", Manifest: manifest}); err != nil {
		t.Fatalf("update manifest failed: %v", err)
	}
	if _, err := client.ExecuteManifest(context.Background(), ClusterCredential{}, ManifestRequest{Action: "delete", Namespace: "default", Manifest: manifest}); err != nil {
		t.Fatalf("delete manifest failed: %v", err)
	}
}

func TestClientGetManifestRedactsSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{{Version: "v1"}})
	mapper.Add(schema.GroupVersionKind{Version: "v1", Kind: "Secret"}, meta.RESTScopeNamespace)
	secret := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      "secret",
			"namespace": "default",
		},
		"data": map[string]interface{}{"password": "c2hvdWxkLW5vdC1sZWFr"},
	}}
	client := Client{DynamicClient: dynamicfake.NewSimpleDynamicClient(scheme, secret), RESTMapper: mapper}

	result, err := client.GetManifest(context.Background(), ClusterCredential{}, ManifestRequest{Namespace: "default", Manifest: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      "secret",
			"namespace": "default",
		},
	}})
	if err != nil {
		t.Fatalf("get secret failed: %v", err)
	}
	object, ok := result["object"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected object map, got=%T", result["object"])
	}
	if object["data"] != nil || object["stringData"] != nil || object["dataRedacted"] != true {
		t.Fatalf("secret object was not redacted: %+v", object)
	}
}

func TestClientExecuteManifestPrunesReadOnlyFields(t *testing.T) {
	scheme := runtime.NewScheme()
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{{Version: "v1"}})
	mapper.Add(schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}, meta.RESTScopeNamespace)
	client := Client{DynamicClient: dynamicfake.NewSimpleDynamicClient(scheme), RESTMapper: mapper}

	result, err := client.ExecuteManifest(context.Background(), ClusterCredential{}, ManifestRequest{Action: "create", Namespace: "default", Manifest: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":              "app-config",
			"namespace":         "default",
			"uid":               "must-be-pruned",
			"resourceVersion":   "must-be-pruned",
			"managedFields":     []interface{}{map[string]interface{}{"manager": "kubectl"}},
			"creationTimestamp": "2026-05-11T00:00:00Z",
		},
		"data":   map[string]interface{}{"APP_ENV": "dev"},
		"status": map[string]interface{}{"phase": "Ready"},
	}})
	if err != nil {
		t.Fatalf("create configmap failed: %v", err)
	}
	object, ok := result["object"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected object map, got=%T", result["object"])
	}
	metadata, _ := object["metadata"].(map[string]interface{})
	if object["status"] != nil || metadata["managedFields"] != nil || metadata["uid"] != nil || metadata["resourceVersion"] != nil {
		t.Fatalf("read-only fields were not pruned: %+v", object)
	}
}

func TestClientExecuteActionDryRunWithFakeClient(t *testing.T) {
	replicas := int32(1)
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"}, Spec: appsv1.DeploymentSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "nginx"}}}}}})
	result, err := (Client{Clientset: clientset}).ExecuteAction(context.Background(), ClusterCredential{}, ActionRequest{Namespace: "default", Kind: "Deployment", Name: "web", Action: "restart", DryRun: true})
	if err != nil {
		t.Fatalf("dry-run restart failed: %v", err)
	}
	if result["dryRun"] != true {
		t.Fatalf("expected dryRun result true, got=%v", result["dryRun"])
	}
}

func TestClientExecuteActionRejectsUnsupportedConfigMapApply(t *testing.T) {
	_, err := (Client{Clientset: fake.NewSimpleClientset()}).ExecuteAction(context.Background(), ClusterCredential{}, ActionRequest{Namespace: "default", Kind: "ConfigMap", Name: "app", Action: "apply", DryRun: true})
	if err == nil {
		t.Fatalf("expected configmap apply to be rejected until manifest support exists")
	}
}

func TestReplicasParamRejectsInvalidValues(t *testing.T) {
	cases := []map[string]interface{}{
		nil,
		{},
		{"replicas": -1},
		{"replicas": 1.2},
		{"replicas": "1.2"},
		{"replicas": int64(1 << 40)},
	}
	for _, item := range cases {
		if _, err := replicasParam(item); err == nil {
			t.Fatalf("expected replicasParam to reject %#v", item)
		}
	}
	if replicas, err := replicasParam(map[string]interface{}{"replicas": "3"}); err != nil || replicas != 3 {
		t.Fatalf("expected string replicas to parse, replicas=%d err=%v", replicas, err)
	}
}
