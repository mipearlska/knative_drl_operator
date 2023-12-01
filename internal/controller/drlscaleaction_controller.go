/*
Copyright 2023 mipearlska.

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

package controller

import (
	"context"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"

	drlscalingv1 "github.com/mipearlska/knative_drl_operator/api/v1"
)

var (
	loggerSD = ctrl.Log.WithName("ControllerLOG")
)

// DRLScaleActionReconciler reconciles a DRLScaleAction object
type DRLScaleActionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=drlscaling.knativescaling.dcn.ssu.ac.kr,resources=drlscaleactions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=drlscaling.knativescaling.dcn.ssu.ac.kr,resources=drlscaleactions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=drlscaling.knativescaling.dcn.ssu.ac.kr,resources=drlscaleactions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DRLScaleAction object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *DRLScaleActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// GetDRLScaleAction resource
	var DRLScaleActionCRD = drlscalingv1.DRLScaleAction{}
	if err := r.Get(ctx, req.NamespacedName, &DRLScaleActionCRD); err != nil {
		loggerSD.Error(err, "unable to fetch DRLScaleCRD")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {
		loggerSD.Info("Fetched CRD sucessfully")
	}

	// Get Scaling Configuration for each service from CRD
	SV_House_Resource := DRLScaleActionCRD.Spec.ServiceHouse_Resource
	SV_House_Concurrency := DRLScaleActionCRD.Spec.ServiceHouse_Concurrency
	SV_House_PodNum := DRLScaleActionCRD.Spec.ServiceSenti_Podcount

	// SV_Senti_Resource := DRLScaleActionCRD.Spec.ServiceSenti_Resource
	// SV_Senti_Concurrency := DRLScaleActionCRD.Spec.ServiceSenti_Concurrency
	// SV_Senti_PodNum := DRLScaleActionCRD.Spec.ServiceSenti_Podcount

	// SV_Numbr_Resource := DRLScaleActionCRD.Spec.ServiceNumbr_Resource
	// SV_Numbr_Concurrency := DRLScaleActionCRD.Spec.ServiceNumbr_Concurrency
	// SV_Numbr_PodNum := DRLScaleActionCRD.Spec.ServiceNumbr_Podcount

	Action_Valid := DRLScaleActionCRD.Spec.ActionValid

	// Initialize Knative Serving Go Client
	// Ref:https://stackoverflow.com/questions/66199455/list-service-in-go
	// This Testbed's MasterNode kubeconfig path = "/root/.kube/config"

	config, err := clientcmd.BuildConfigFromFlags("", "/root/.kube/config")
	if err != nil {
		loggerSD.Error(err, "unable to BuildConfigFromFlags using clientcmd")
	}

	serving, err := servingv1client.NewForConfig(config)
	if err != nil {
		loggerSD.Error(err, "unable to create Knative Serving Go Client")
	}

	//**Get Service with name == TrafficStatCRD.spec.servicename
	ServiceHouse, err := serving.Services("default").Get(ctx, "deploy-a", metav1.GetOptions{})
	if err != nil {
		loggerSD.Info("TargetService name from CRD is:", "SERVICE_NAME", "deploy-a")
		loggerSD.Error(err, "TargetService from CRD is not available in cluster")
	} else {
		loggerSD.Info("TargetService name from CRD is:", "SERVICE_NAME", "deploy-a")
		loggerSD.Info("Found TargetService in cluster:", "SERVICE_NAME", ServiceHouse.Name)
	}

	ServiceHouse_Current_Revision := ServiceHouse.Status.LatestReadyRevisionName

	// Create New Service_House Configuration
	var NewServiceHouseConfiguration *servingv1.Service

	if Action_Valid == "1" {

		NewServiceHouseConfiguration = &servingv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-a",
				Namespace: "default",
				Labels: map[string]string{
					"app": "deploy-a",
				},
				Annotations: map[string]string{
					"serving.knative.dev/creator":      "kubernetes-admin",
					"serving.knative.dev/lastModifier": "kubernetes-admin",
				},
			},
			Spec: servingv1.ServiceSpec{
				ConfigurationSpec: servingv1.ConfigurationSpec{
					Template: servingv1.RevisionTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "deploy-a",
							},
							Annotations: map[string]string{
								"autoscaling.knative.dev/target":        SV_House_Concurrency,
								"autoscaling.knative.dev/initial-scale": SV_House_PodNum,
								"autoscaling.knative.dev/min-scale":     SV_House_PodNum,
							},
						},
						Spec: servingv1.RevisionSpec{
							PodSpec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "deploy-a",
										Image: "vudinhdai2505/test-app:v5",
										Resources: corev1.ResourceRequirements{
											Limits: map[corev1.ResourceName]resource.Quantity{
												"cpu": resource.MustParse(SV_House_Resource),
											},
										},
										Ports: []corev1.ContainerPort{
											{
												ContainerPort: 5000,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

	} else if Action_Valid == "0" {

		NewServiceHouseConfiguration = &servingv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deploy-a",
				Namespace: "default",
				Labels: map[string]string{
					"app": "deploy-a",
				},
				Annotations: map[string]string{
					"serving.knative.dev/creator":      "kubernetes-admin",
					"serving.knative.dev/lastModifier": "kubernetes-admin",
				},
			},
			Spec: servingv1.ServiceSpec{
				ConfigurationSpec: servingv1.ConfigurationSpec{
					Template: servingv1.RevisionTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "deploy-a",
							},
							Annotations: map[string]string{
								"autoscaling.knative.dev/target":        SV_House_Concurrency,
								"autoscaling.knative.dev/initial-scale": SV_House_PodNum,
								"autoscaling.knative.dev/min-scale":     SV_House_PodNum,
							},
						},
						Spec: servingv1.RevisionSpec{
							PodSpec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "deploy-a",
										Image: "vudinhdai2505/test-app:v5",
										Resources: corev1.ResourceRequirements{
											Limits: map[corev1.ResourceName]resource.Quantity{
												"cpu": resource.MustParse(SV_House_Resource),
											},
										},
										Ports: []corev1.ContainerPort{
											{
												ContainerPort: 5000,
											},
										},
									},
								},
								NodeSelector: map[string]string{
									"deploystatus": "mainnode",
								},
							},
						},
					},
				},
			},
		}
	}

	loggerSD.Info("Creating new Configuration for service ", "SERVICE_NAME", "deploy-a")
	loggerSD.Info("with chosen settings ", "CHOSEN_RESOURCE_LEVEL", SV_House_Resource)
	loggerSD.Info("with chosen settings ", "CHOSEN_CONCURRENCY", SV_House_Concurrency)

	//// Set ResourceVersion of new Configuration to the current Service's ResourceVersion (Required for Update)
	NewServiceHouseConfiguration.SetResourceVersion(ServiceHouse.GetResourceVersion())

	//// Call KnativeServingClient to create new Service Revision by updating current service with new Configuration
	NewServiceRevision, err := serving.Services("default").Update(ctx, NewServiceHouseConfiguration, metav1.UpdateOptions{})

	// New Revision Number = current + 1 (from service-00009 to service-00010) (Below are string processing to get the new Revision ID/Number)
	tempstring := strings.Split(ServiceHouse_Current_Revision, "-")
	tempint, _ := strconv.Atoi(tempstring[len(tempstring)-1])
	rev_number := strconv.Itoa(tempint + 1)
	New_Revision_Number := "deploy-a" + "-" + strings.Repeat("0", 5-len(rev_number)) + rev_number
	if err != nil {
		loggerSD.Error(err, err.Error())
	} else {
		loggerSD.Info("New Service Revision Created", "SERVICE", NewServiceRevision.Name)
		loggerSD.Info("New Service Revision Number", "REV_NUMBER", New_Revision_Number)

		// Watch New Revision,
		// Wait until new Revision ready (Pod Running)
		// Delete old Revision and the corresponding pods (to handle previous Revision long Terminating pods time, which can hold a lot of worker node resources)

		// While Loop to wait until New Revision Pod Ready to serve
		for {
			time.Sleep(1 * time.Second)
			BeforeDeleteRevisionPodList := &corev1.PodList{}
			if err := r.List(ctx, BeforeDeleteRevisionPodList); err != nil {
				loggerSD.Error(err, err.Error())
				break
			}
			count := 0
			newPodDeploy := false
			for _, pod := range BeforeDeleteRevisionPodList.Items {
				if strings.HasPrefix(pod.Name, New_Revision_Number) && pod.Status.Phase == "Running" {
					newPodDeploy = true // New Pod Not Ready, keep previous Revision alive
					count += 1
				} else if strings.HasPrefix(pod.Name, New_Revision_Number) && pod.Status.Phase != "Running" {
					newPodDeploy = false
				}
			}
			if count == 0 || !newPodDeploy {
				loggerSD.Info("New Revision Pod NOT READY", "REV_NUMBER", New_Revision_Number)
			} else { // only when New Revision Pod Ready, process to Delete Previous Revision Pods step
				loggerSD.Info("New Revision Pod Running")
				break
			}
		}

		loggerSD.Info("Wait")
		time.Sleep(5 * time.Second)

		// New Revision Pods are READY now, Delete old Revision and old Revision pods
		// Check if old Revision Pods are still Terminating. If YES delete old Revision, Then Delete pod
		ReadyDeleteRevisionPodList := &corev1.PodList{}
		if err := r.List(ctx, ReadyDeleteRevisionPodList); err != nil {
			loggerSD.Error(err, err.Error())
		} else {
			count := 0 // count to ensure Delete Revision is only called one time in the PodList loop (when count = 1)
			for _, pod := range ReadyDeleteRevisionPodList.Items {
				if strings.HasPrefix(pod.Name, ServiceHouse_Current_Revision) {
					targetpod := &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      pod.Name,
						},
					}
					count += 1
					if count == 1 {
						loggerSD.Info("Ask to delete Revision", "REVISION_NAME", ServiceHouse_Current_Revision)

						err := serving.Revisions("default").Delete(context.Background(), ServiceHouse_Current_Revision, metav1.DeleteOptions{})
						if err != nil {
							loggerSD.Error(err, err.Error())
						} else {
							loggerSD.Info("Delete Revision ", "REVISION_NAME", ServiceHouse_Current_Revision)
						}
						time.Sleep(2 * time.Second)
					}

					if err := r.Delete(ctx, targetpod, client.GracePeriodSeconds(0)); err != nil {
						loggerSD.Error(err, err.Error())
					} else {
						loggerSD.Info("Delete pod ", "POD_NAME", pod.Name)
					}
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DRLScaleActionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&drlscalingv1.DRLScaleAction{}).
		Complete(r)
}
