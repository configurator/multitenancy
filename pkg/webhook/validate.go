package webhook

import (
	"encoding/json"
	"fmt"
	"reflect"

	v1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	tenantKind = reflect.TypeOf(v1.Tenant{}).Name()
)

// validate logs the incoming request and passes its parameters to the appropriate
// validation function.
func (s *webhookServer) validate(ar *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	req := ar.Request

	webhooklog.Info(fmt.Sprintf("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo))

	// TODO: Check for enum
	if req.Operation == "DELETE" {
		return s.validateDelete(req)
	}
	return s.validateApply(req)
}

// validateApply checks whether a provided create/update request is valid and should
// be allowed.
func (s *webhookServer) validateApply(req *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	switch req.Kind.Kind {

	case tenantKind:
		tenant := &v1.Tenant{}
		if err := json.Unmarshal(req.Object.Raw, &tenant); err != nil {
			webhooklog.Error(err, "Could not unmarshal raw object")
			return notAllowed(err.Error(), metav1.StatusReasonBadRequest)
		}
		return s.validateTenant(tenant)

	default:
		return notAllowed(fmt.Sprintf("Unexpected resource kind: %s", req.Kind.Kind), metav1.StatusReasonBadRequest)
	}
}

// validateDelete is placeholder code for validating deletion operations. In the
// future this could be used for things like deletion protection.
func (s *webhookServer) validateDelete(req *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	// Note for any future DELETE validation that the request object is empty and we only have the name/namespace
	return &admissionv1beta1.AdmissionResponse{
		Allowed: true,
	}
}

// validateTenant returns whether the provided tenant spec is valid and should be allowed.
func (s *webhookServer) validateTenant(tenant *v1.Tenant) *admissionv1beta1.AdmissionResponse {
	// Make sure the multitenancy exists
	if _, err := tenant.GetTenancy(s.client); err != nil {
		if kerrors.IsNotFound(err) {
			return notAllowed(fmt.Sprintf("There is no tenancy kind %s in namespace %s", tenant.TenancyKind, tenant.Namespace), metav1.StatusReasonNotFound)
		}
		return notAllowed(fmt.Sprintf("Unexpected API error during request: %s", err.Error()), metav1.StatusReasonBadRequest)
	}
	return &admissionv1beta1.AdmissionResponse{
		Allowed: true,
	}
}
