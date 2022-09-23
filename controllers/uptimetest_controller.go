/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/StatusCakeDev/statuscake-go"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	srev1alpha1 "https://github.com/sharkymcdongles/statuscake-operator/api/v1alpha1"
)

// UptimeTestReconciler reconciles a UptimeTest object
type UptimeTestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sre.mls.io,resources=uptimetests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sre.mls.io,resources=uptimetests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sre.mls.io,resources=uptimetests/finalizers,verbs=update
func (r *UptimeTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Create statuscake client
	c := StatusCakeClient(os.Getenv("STATUSCAKE_API_TOKEN"))

	// Set ctrl.Result{} to var because it is prettier that way
	result := ctrl.Result{}
	// Fetch the UptimeTest instance
	uptimeTest := &srev1alpha1.UptimeTest{}

	// get api group
	gvk := srev1alpha1.GroupVersion.WithKind("uptimetest")
	gk := gvk.Kind + "." + gvk.Group
	// create an annotation to track the test id for uptimetests
	uptimeTestAnnotation := gk + "/statuscake-test-id"
	// create a finalizer to ensure deletion of uptimetests via the statuscake api upon deletion of the custom resource
	uptimeTestFinalizer := gk + "/finalizer"

	if err := r.Get(ctx, req.NamespacedName, uptimeTest); err != nil {
		// NotFound cannot be fixed by requeuing so ignore it.
		// Return and don't requeue
		if err = client.IgnoreNotFound(err); err != nil {
			log.Log.Error(err, "failed to get uptimetest")
		}
		// Error reading the object - requeue the request.
		return result, err
	}

	// Keep a copy of cluster prior to any manipulations.
	uptimeTestBefore := uptimeTest.DeepCopy()
	// Get all currently set annotations for the UptimeTest CRD
	uptimeTestAnnotationsMap := uptimeTest.GetAnnotations()
	// Create an empty map[string]string if no annotations currently exist
	if uptimeTestAnnotationsMap == nil {
		uptimeTestAnnotationsMap = make(map[string]string)
	}

	if uptimeTest.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(uptimeTest, uptimeTestFinalizer) {
			controllerutil.AddFinalizer(uptimeTest, uptimeTestFinalizer)
			log.Log.Info("adding the finalizer to uptimetest")
			if err := r.Update(ctx, uptimeTest); err != nil {
				log.Log.Error(err, "something went wrong when adding the finalizer to uptimetest")
				return result, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(uptimeTest, uptimeTestFinalizer) && uptimeTestAnnotationsMap[uptimeTestAnnotation] != "" {
			// call statuscake API to delete the test if the finalizer is present
			log.Log.Info("calling statuscake api to delete the uptimetest due to finalizer")
			if err := DeleteUptimeTest(c, uptimeTestAnnotationsMap[uptimeTestAnnotation]); err != nil {
				log.Log.Error(err, "something went wrong when deleting the uptimetest via the statuscake api")
				// if error retry the request
				return result, err
			}

			// remove finalizer from list and update it
			controllerutil.RemoveFinalizer(uptimeTest, uptimeTestFinalizer)
			log.Log.Info("finalizer removed from the uptimetest to allow deletion")
			if err := r.Update(ctx, uptimeTest); err != nil {
				log.Log.Error(err, "something went wrong when removing the finalizer from the uptimetest")
				return result, err
			}
			return result, nil
		}
	}

	// Define the function for the updating the uptimetest status
	patchUptimeTest := func() (reconcile.Result, error) {
		if !equality.Semantic.DeepEqual(uptimeTestBefore.Status, uptimeTest.Status) {
			// NOTE: Kubernetes prior to v1.16.10 and v1.17.6 does not track
			// managed fields on the status subresource: https://issue.k8s.io/88901
			if err := errors.WithStack(r.Client.Status().Patch(
				ctx, uptimeTest, client.MergeFrom(uptimeTestBefore))); err != nil {
				log.Log.Error(err, "patching uptimetest status")
				return result, err
			}
			log.Log.Info("patched uptimetest status")
		}
		return result, nil
	}

	// Create UptimeTestAPISpec for use with the StatusCake API
	checkName := fmt.Sprintf("%s-%s", uptimeTest.GetName(), uptimeTest.GetNamespace())
	uptimeTestAPISpec := NewUptimeTestAPISpecFromCRD(uptimeTest.Spec, checkName)

	// Create new test if the annotation doesn't exist and then set the annotation to prevent duplication of checks
	if uptimeTestAnnotationsMap[uptimeTestAnnotation] == "" {
		uptimeTestAPIResponse, err := CreateUptimeTest(c, uptimeTestAPISpec)
		if err != nil {
			log.Log.Error(err, "something went wrong when creating the uptimetest via the statuscake api")
			if len(statuscake.Errors(err)) > 0 {
				log.Log.Info(fmt.Sprintf("errors output: %v", statuscake.Errors(err)))
			}
		}
		log.Log.Info(fmt.Sprintf("new test created with id: %v", uptimeTestAPIResponse.Data.NewID))
		uptimeTestAnnotationsMap[uptimeTestAnnotation] = uptimeTestAPIResponse.Data.NewID
		uptimeTest.SetAnnotations(uptimeTestAnnotationsMap)

		meta.SetStatusCondition(&uptimeTest.Status.Conditions, metav1.Condition{
			Message:            "UptimeTest created successfully",
			Type:               srev1alpha1.Created,
			Status:             metav1.ConditionTrue,
			Reason:             "CustomResourceCreated",
			ObservedGeneration: uptimeTest.GetGeneration(),
		})

		err = r.Update(ctx, uptimeTest)
		if err != nil {
			log.Log.Error(err, "failed to annotate uptimetest ", "namespace: ", uptimeTest.Namespace, "name: ", uptimeTest.Name)
			return result, err
		}

		return patchUptimeTest()

	} else {
		log.Log.Info(fmt.Sprintf("existing test found with id: %v", uptimeTest.Annotations[uptimeTestAnnotation]))
		existingUptimeTestID := uptimeTest.Annotations[uptimeTestAnnotation]
		existingUptimeTestAPISpec, err := GetUptimeTest(c, existingUptimeTestID)

		if err != nil {
			log.Log.Error(err, "something went wrong when retrieving the uptimetest via the statuscake api")
			if len(statuscake.Errors(err)) > 0 {
				log.Log.Info(fmt.Sprintf("errors output: %v", statuscake.Errors(err)))
			}
			return result, err
		}

		if reflect.DeepEqual(existingUptimeTestAPISpec, uptimeTestAPISpec) {
			log.Log.Info("expected state matches desired state. nothing to be done")
			// If the test matches the spec do nothing
			return result, nil

		} else {
			log.Log.Info("updating uptimetest with desired state")
			// Update the existing test with the changes from the CRD
			err := UpdateUptimeTest(c, existingUptimeTestID, uptimeTestAPISpec)

			if err != nil {
				// Error reading the object - requeue the request.
				log.Log.Error(err, "failed to update uptimetest")
				if len(statuscake.Errors(err)) > 0 {
					log.Log.Info(fmt.Sprintf("errors output: %v", statuscake.Errors(err)))
				}
				return result, err
			}
			meta.SetStatusCondition(&uptimeTest.Status.Conditions, metav1.Condition{
				Message:            "UptimeTest updated successfully",
				Type:               srev1alpha1.Updated,
				Status:             metav1.ConditionTrue,
				Reason:             "CustomResourceSpecUpdated",
				ObservedGeneration: uptimeTest.GetGeneration(),
			})

			return patchUptimeTest()
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *UptimeTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&srev1alpha1.UptimeTest{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
