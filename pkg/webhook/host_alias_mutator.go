package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const dnsAnnotationKey = "percona.com/dns-zone"
const loopback = "127.0.0.1"

type HostAliasMutator struct {
	decoder *admission.Decoder
}

func (a *HostAliasMutator) Handle(_ context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)

	if err != nil {
		klog.Errorf("Failed to decode request: %s", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil || pod.Annotations[dnsAnnotationKey] == "" {
		klog.Infof("%s annotation not found, skipping", dnsAnnotationKey)
		return admission.Allowed("nop")
	}

	if pod.Spec.HostNetwork {
		klog.Warningf("pod/%s in namespace %s uses host network which excludes host alias usage", pod.Name, pod.Namespace)
		return admission.Errored(http.StatusBadRequest, errors.New("host network and DNS annotation are mutually exclusive"))
	}

	externalFqdn := fmt.Sprintf("%s.%s", pod.Name, pod.Annotations[dnsAnnotationKey])

	klog.Infof("Appending new host alias %s to pod/%s in namespace %s", externalFqdn, pod.Name, pod.Namespace)

	pod.Spec.HostAliases = append(pod.Spec.HostAliases, corev1.HostAlias{IP: loopback, Hostnames: []string{externalFqdn}})

	podJson, err := json.Marshal(pod)

	if err != nil {
		klog.Errorf("Error marshaling pod: %s", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, podJson)
}

func (a *HostAliasMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
