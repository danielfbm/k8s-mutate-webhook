// Package mutate deals with AdmissionReview requests and responses, it takes in the request body and returns a readily converted JSON []byte that can be
// returned from a http Handler w/o needing to further convert or modify it, it also makes testing Mutate() kind of easy w/o need for a fake http server, etc.
package mutate

import (
	"encoding/json"
	"fmt"
	"log"

	v1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mutate mutates
func Mutate(body []byte, verbose bool) ([]byte, error) {
	if verbose {
		log.Printf("recv: %s\n", string(body)) // untested section
	}

	// unmarshal request into AdmissionReview struct
	admReview := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	var err error
	// var pod *corev1.Pod
	var pvc *corev1.PersistentVolumeClaim

	responseBody := []byte{}
	ar := admReview.Request
	resp := v1beta1.AdmissionResponse{}

	if ar != nil {

		// get the Pod object and unmarshal it into its struct, if we cannot, we might as well stop here
		if err := json.Unmarshal(ar.Object.Raw, &pvc); err != nil {
			return nil, fmt.Errorf("unable unmarshal pvc json object %v", err)
		}

		log.Println("object request pvc", pvc)
		// set response options
		resp.Allowed = true
		resp.UID = ar.UID
		pT := v1beta1.PatchTypeJSONPatch
		resp.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

		// add some audit annotations, helpful to know why a object was modified, maybe (?)
		resp.AuditAnnotations = map[string]string{
			"mutateme": "yup it did it",
		}

		storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		oneGbi := resource.MustParse("1Gi")
		if ok && storage.Cmp(oneGbi) <= -1 {
			// if it is lower then 1Gi
			// the actual mutation is done by a string in JSONPatch style, i.e. we don't _actually_ modify the object, but
			// tell K8S how it should modifiy it
			p := []map[string]string{}
			patch := map[string]string{
				"op":    "replace",
				"path":  "/spec/resources/requests/storage",
				"value": "1Gi",
			}
			p = append(p, patch)
			resp.Patch, err = json.Marshal(p)

			// Success, of course ;)
			resp.Result = &metav1.Status{
				Status: "Success",
			}
		} else {
			resp.Allowed = true
		}

		// for i := range pod.Spec.Containers {
		// 	patch := map[string]string{
		// 		"op":    "replace",
		// 		"path":  fmt.Sprintf("/spec/containers/%d/image", i),
		// 		"value": "debian",
		// 	}
		// 	p = append(p, patch)
		// }
		// parse the []map into JSON
		// resp.Patch, err = json.Marshal(p)

		// Success, of course ;)
		// resp.Result = &metav1.Status{
		// 	Status: "Success",
		// }

		admReview.Response = &resp
		// back into JSON so we can return the finished AdmissionReview w/ Response directly
		// w/o needing to convert things in the http handler
		responseBody, err = json.Marshal(admReview)
		if err != nil {
			return nil, err // untested section
		}
	}

	if verbose {
		log.Printf("resp: %s\n", string(responseBody)) // untested section
	}

	return responseBody, nil
}
