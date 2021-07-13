package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const dnsAnnotationKey = "percona.com/dns-zone"
const loopback = "127.0.0.1"

var log = logf.Log.WithName("host-alias-mutator")

type HostAliasMutator struct {
	decoder *admission.Decoder
}

func (a *HostAliasMutator) Handle(ctx context.Context, req admission.Request) admission.Response{
	pod := &corev1.Pod{}
	
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "some-pod",
			Namespace:   "some-namespace",
			Annotations: map[string]string{"foo": "bar", "x": "y"},
		},
		Spec: corev1.PodSpec{
			HostAliases: []corev1.HostAlias{{IP: "127.0.0.1", Hostnames: []string{"foo.example.com"}}},
		},
	}

	testPodJson, err2 := json.Marshal(testPod)

	if err2 != nil {
		panic(err2)
	}

	err2 = a.decoder.Decode(admission.Request{AdmissionRequest: v1beta1.AdmissionRequest{
		UID:                "some-uid",
		Kind:               metav1.GroupVersionKind{},
		Resource:           metav1.GroupVersionResource{},
		Object:             runtime.RawExtension{Raw: testPodJson},
		OldObject:          runtime.RawExtension{},
		DryRun:             nil,
		Options:            runtime.RawExtension{},
	}}, testPod)

	err := a.decoder.Decode(req, pod)

	if err != nil {
		log.Error(err, "Failed to decode request")
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil || pod.Annotations[dnsAnnotationKey] == "" {
		log.Info(fmt.Sprintf("%s annotation not found, skipping", dnsAnnotationKey))
		return admission.Allowed("nop")
	}

	externalFqdn := fmt.Sprintf("%s.%s", pod.Name, pod.Annotations[dnsAnnotationKey])
	externalFqdnAdded := false
	for _, alias := range pod.Spec.HostAliases {
		if alias.IP == loopback {
			log.Info("Extending existing host alias", "alias", externalFqdn)
			alias.Hostnames = append(alias.Hostnames, externalFqdn)
			externalFqdnAdded = true
			break
		}
	}

	if !externalFqdnAdded {
		log.Info("Appending new host alias", "alias", externalFqdn)
		pod.Spec.HostAliases = append(pod.Spec.HostAliases, corev1.HostAlias{IP: loopback, Hostnames: []string{externalFqdn}})
	}

	podJson, err := json.Marshal(pod)

	if err != nil {
		log.Error(err, "Error marshaling pod")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, podJson)
}

func (a *HostAliasMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
