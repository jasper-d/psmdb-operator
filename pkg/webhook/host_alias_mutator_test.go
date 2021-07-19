package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func TestHostAliasMutator_Handle(t *testing.T) {

	const podName string = "some-pod"
	const namespace string = "some-namespace"

	scheme := runtime.NewScheme()
	decoder, err := admission.NewDecoder(scheme)

	if err != nil {
		t.Error("Failed to create decoder")
	}

	setup := func(annotations map[string]string, aliases []corev1.HostAlias) (*corev1.Pod, admission.Request) {
		testPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        podName,
				Namespace:   namespace,
				Annotations: annotations, // map[string]string{"foo": "bar", "x": "y"},
			},
			Spec: corev1.PodSpec{
				HostAliases: aliases, // []corev1.HostAlias{{IP: "127.0.0.1", Hostnames: []string{"foo.example.com"}}},
			},
		}

		testPodJson, err2 := json.Marshal(testPod)

		if err2 != nil {
			panic(err2)
		}

		req := admission.Request{AdmissionRequest: v1beta1.AdmissionRequest{
			UID:       "some-uid",
			Kind:      metav1.GroupVersionKind{},
			Resource:  metav1.GroupVersionResource{},
			Object:    runtime.RawExtension{Raw: testPodJson},
			OldObject: runtime.RawExtension{},
			DryRun:    nil,
			Options:   runtime.RawExtension{},
		}}

		return testPod, req
	}

	wh := &HostAliasMutator{}
	wh.InjectDecoder(decoder)

	t.Run("Invalid request returns 400", func(t *testing.T) {
		_, req := setup(nil, nil)

		req.Object.Raw = []byte("this isn't valid")

		resp := wh.Handle(context.TODO(), req)

		if resp.Allowed {
			t.Errorf("expected allowed %v, actual %v", false, resp.Allowed)
		}

		if resp.Result.Code != 400 {
			t.Errorf("expected code %v, actual %v", 400, resp.Result.Status)
		}
	})

	dnsZone := "foo.example.com"

	testCases := []struct {
		name               string
		annotations        map[string]string
		aliases            []corev1.HostAlias
		expectedPatchCount int
		expectedAliases    []corev1.HostAlias
	}{
		{
			"all nil",
			nil,
			nil,
			0,
			nil,
		},
		{
			"no op",
			map[string]string{"foo": "bar"},
			nil,
			0,
			nil,
		},
		{
			"new alias",
			map[string]string{dnsAnnotationKey: dnsZone},
			nil,
			1,
			[]corev1.HostAlias{{
				IP:        loopback,
				Hostnames: []string{fmt.Sprintf("%s.%s", "some-pod", dnsZone)},
			}},
		},
		{
			"additional alias",
			map[string]string{dnsAnnotationKey: dnsZone},
			[]corev1.HostAlias{{
				IP:        loopback,
				Hostnames: []string{"localhost"}}},
			1,
			[]corev1.HostAlias{
				{
					IP:        loopback,
					Hostnames: []string{"localhost"},
				},
				{
					IP:        loopback,
					Hostnames: []string{fmt.Sprintf("%s.%s", "some-pod", dnsZone)},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Annotation added to aliases %s", tc.name), func(t *testing.T) {
			client := fake.NewSimpleClientset()
			pod, req := setup(tc.annotations, tc.aliases)
			client.CoreV1().Pods(namespace).Create(pod)
			resp := wh.Handle(context.TODO(), req)

			if !resp.Allowed {
				t.Errorf("expected allowed %v, actual %v", true, resp.Allowed)
			}

			if (resp.Patches == nil && tc.expectedPatchCount > 0) || len(resp.Patches) != tc.expectedPatchCount {
				t.Errorf("expected %v patches, actual %v", 1, len(resp.Patches))
			}

			if len(resp.Patches) == 0 {
				return
			}

			patchJson, err := json.Marshal(resp.Patches)

			if err != nil {
				t.Errorf("expected no err when marshaling patches, got %v", err)
			}

			patchedPod, err := client.CoreV1().Pods(namespace).Patch(pod.Name, types.JSONPatchType, patchJson)

			missing := []string{}
			for _, ea := range tc.expectedAliases {
				for i, aa := range patchedPod.Spec.HostAliases {
					if ea.IP == aa.IP {
						if reflect.DeepEqual(ea.Hostnames, aa.Hostnames) {
							break
						}
					}
					if i == len(patchedPod.Spec.HostAliases)-1 {
						missing = append(missing, fmt.Sprintf("%s, %s", ea.IP, ea.Hostnames[0]))
					}
				}

			}

			if len(missing) > 0 {
				t.Errorf("failed to find hostaliases %v", missing)
			}
		})
	}
}
