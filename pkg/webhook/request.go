package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// serve is the main entrypoint for requests coming in on the validate path.
// The request is deserialized, passed to the validators, and then a response
// is returned to the requesting client.
func (s *webhookServer) serve(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		webhooklog.Error(err, "Failed to read request body")
		http.Error(w, "could not read request body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		webhooklog.Error(errors.New("empty body"), "empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		err = fmt.Errorf("Content-Type=%s, expect application/json", contentType)
		webhooklog.Error(err, "invalid content type")
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *admissionv1beta1.AdmissionResponse
	ar := admissionv1beta1.AdmissionReview{}
	if _, _, err := s.deserializer.Decode(body, nil, &ar); err != nil {
		webhooklog.Error(err, "Can't decode body")
		admissionResponse = notAllowed(err.Error(), metav1.StatusReasonBadRequest)
	} else {
		admissionResponse = s.validate(&ar)
	}

	admissionReview := admissionv1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		webhooklog.Error(err, "Can't encode response")
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	if _, err := w.Write(resp); err != nil {
		webhooklog.Error(err, "Can't write response")
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}

}

// notAllowed returns an admission response that tells the api servers
// to reject the resource change for the given reason. The msg is returned back to
// the requesting party (e.g kubectl, client-api POSTs).
func notAllowed(msg string, reason metav1.StatusReason) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: msg,
			Reason:  reason,
		},
	}
}
