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
	"fmt"
	"strconv"
	"strings"

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

	sv_house_name := "deploy-a"
	sv_senti_name := "sentiment"
	sv_numbr_name := "numberreg"

	required_change_service_array := []string{}

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
	SV_House_PodNum := DRLScaleActionCRD.Spec.ServiceHouse_Podcount

	SV_Senti_Resource := DRLScaleActionCRD.Spec.ServiceSenti_Resource
	SV_Senti_Concurrency := DRLScaleActionCRD.Spec.ServiceSenti_Concurrency
	SV_Senti_PodNum := DRLScaleActionCRD.Spec.ServiceSenti_Podcount

	SV_Numbr_Resource := DRLScaleActionCRD.Spec.ServiceNumbr_Resource
	SV_Numbr_Concurrency := DRLScaleActionCRD.Spec.ServiceNumbr_Concurrency
	SV_Numbr_PodNum := DRLScaleActionCRD.Spec.ServiceNumbr_Podcount

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

	//**Get current Service House Object + Info (Revision, Con, Res)
	ServiceHouse, err := serving.Services("default").Get(ctx, sv_house_name, metav1.GetOptions{})
	if err != nil {
		loggerSD.Error(err, "TargetService from CRD is not available in cluster")
	} else {
		loggerSD.Info("Found TargetService in cluster:", "SERVICE_NAME", ServiceHouse.Name)
	}

	ServiceHouse_Current_Revision := ServiceHouse.Status.LatestReadyRevisionName
	ServiceHouse_Current_Con := ServiceHouse.Spec.Template.ObjectMeta.Annotations["autoscaling.knative.dev/target"]
	ServiceHouse_Current_Res := ServiceHouse.Spec.Template.Spec.Containers[0].Resources.Limits["cpu"]

	//**If different scaling configuration required, Create new service House Configuration
	var NewServiceHouseConfiguration *servingv1.Service
	var NewServiceHouseRevision *servingv1.Service
	var NewServiceHouseRevNumber string
	if SV_House_Concurrency == ServiceHouse_Current_Con && SV_House_Resource == ConvertResourceLimitToString(ServiceHouse_Current_Res) {
		loggerSD.Info("No change required for service House")
	} else {
		required_change_service_array = append(required_change_service_array, "deploy-a")
		NewServiceHouseConfiguration = CreateNewSVHouseConfiguration(SV_House_Concurrency, SV_House_Resource, SV_House_PodNum)
		NewServiceHouseConfiguration.SetResourceVersion(ServiceHouse.GetResourceVersion())
		loggerSD.Info("New Configuration ", "house", SV_House_Concurrency, "-", SV_House_Resource, "-", SV_House_PodNum)

		NewServiceHouseRevision, err = serving.Services("default").Update(ctx, NewServiceHouseConfiguration, metav1.UpdateOptions{})
		NewServiceHouseRevNumber = CalculateNewRevisionNumber("deploy-a", ServiceHouse_Current_Revision)
		if err != nil {
			loggerSD.Error(err, err.Error())
		} else {
			loggerSD.Info("New Service Revision Created", "SERVICE", NewServiceHouseRevision.Name)
			loggerSD.Info("New Service Revision Number", "REV_NUMBER", NewServiceHouseRevNumber)
		}
	}

	//**Get current Service Senti Object + Info (Revision, Con, Res)
	ServiceSenti, err := serving.Services("default").Get(ctx, sv_senti_name, metav1.GetOptions{})
	if err != nil {
		loggerSD.Error(err, "TargetService from CRD is not available in cluster")
	} else {
		loggerSD.Info("Found TargetService in cluster:", "SERVICE_NAME", ServiceSenti.Name)
	}

	ServiceSenti_Current_Revision := ServiceSenti.Status.LatestReadyRevisionName
	ServiceSenti_Current_Con := ServiceSenti.Spec.Template.ObjectMeta.Annotations["autoscaling.knative.dev/target"]
	ServiceSenti_Current_Res := ServiceSenti.Spec.Template.Spec.Containers[0].Resources.Limits["cpu"]

	//**If different scaling configuration required, Create new service Senti Configuration
	var NewServiceSentiConfiguration *servingv1.Service
	var NewServiceSentiRevision *servingv1.Service
	var NewServiceSentiRevNumber string
	if SV_Senti_Concurrency == ServiceSenti_Current_Con && SV_Senti_Resource == ConvertResourceLimitToString(ServiceSenti_Current_Res) {
		loggerSD.Info("No change required for service Senti")
	} else {
		required_change_service_array = append(required_change_service_array, "sentiment")
		NewServiceSentiConfiguration = CreateNewSVSentiConfiguration(SV_Senti_Concurrency, SV_Senti_Resource, SV_Senti_PodNum)
		NewServiceSentiConfiguration.SetResourceVersion(ServiceSenti.GetResourceVersion())
		loggerSD.Info("New Configuration ", "senti", SV_Senti_Concurrency, "-", SV_Senti_Resource, "-", SV_Senti_PodNum)

		NewServiceSentiRevision, err = serving.Services("default").Update(ctx, NewServiceSentiConfiguration, metav1.UpdateOptions{})
		NewServiceSentiRevNumber = CalculateNewRevisionNumber("sentiment", ServiceSenti_Current_Revision)
		if err != nil {
			loggerSD.Error(err, err.Error())
		} else {
			loggerSD.Info("New Service Revision Created", "SERVICE", NewServiceSentiRevision.Name)
			loggerSD.Info("New Service Revision Number", "REV_NUMBER", NewServiceSentiRevNumber)
		}
	}

	//**Get current Service Numbr Object + Info (Revision, Con, Res)
	ServiceNumbr, err := serving.Services("default").Get(ctx, sv_numbr_name, metav1.GetOptions{})
	if err != nil {
		loggerSD.Error(err, "TargetService from CRD is not available in cluster")
	} else {
		loggerSD.Info("Found TargetService in cluster:", "SERVICE_NAME", ServiceNumbr.Name)
	}

	ServiceNumbr_Current_Revision := ServiceNumbr.Status.LatestReadyRevisionName
	ServiceNumbr_Current_Con := ServiceNumbr.Spec.Template.ObjectMeta.Annotations["autoscaling.knative.dev/target"]
	ServiceNumbr_Current_Res := ServiceNumbr.Spec.Template.Spec.Containers[0].Resources.Limits["cpu"]

	//**If different scaling configuration required, Create new service Numbr Configuration
	var NewServiceNumbrConfiguration *servingv1.Service
	var NewServiceNumbrRevision *servingv1.Service
	var NewServiceNumbrRevNumber string
	if SV_Numbr_Concurrency == ServiceNumbr_Current_Con && SV_Numbr_Resource == ConvertResourceLimitToString(ServiceNumbr_Current_Res) {
		loggerSD.Info("No change required for service Numbr")
	} else {
		required_change_service_array = append(required_change_service_array, "numberreg")
		NewServiceNumbrConfiguration = CreateNewSVNumbrConfiguration(SV_Numbr_Concurrency, SV_Numbr_Resource, SV_Numbr_PodNum)
		NewServiceNumbrConfiguration.SetResourceVersion(ServiceNumbr.GetResourceVersion())
		loggerSD.Info("New Configuration ", "numbr", SV_Numbr_Concurrency, "-", SV_Numbr_Resource, "-", SV_Numbr_PodNum)

		NewServiceNumbrRevision, err = serving.Services("default").Update(ctx, NewServiceNumbrConfiguration, metav1.UpdateOptions{})
		NewServiceNumbrRevNumber = CalculateNewRevisionNumber("numberreg", ServiceNumbr_Current_Revision)
		if err != nil {
			loggerSD.Error(err, err.Error())
		} else {
			loggerSD.Info("New Service Revision Created", "SERVICE", NewServiceNumbrRevision.Name)
			loggerSD.Info("New Service Revision Number", "REV_NUMBER", NewServiceNumbrRevNumber)
		}

	}
	for _, item := range required_change_service_array {
		loggerSD.Info(item)
	}

	//// Set ResourceVersion of new Configuration to the current Service's ResourceVersion (Required for Update)

	//// Call KnativeServingClient to create new Service Revision by updating current service with new Configuration

	// Watch New Revision,
	// Wait until new Revision ready (Pod Running)
	// Delete old Revision and the corresponding pods (to handle previous Revision long Terminating pods time, which can hold a lot of worker node resources)

	// While Loop to wait until New Revision Pod Ready to serve

	// for {
	// 	time.Sleep(1 * time.Second)
	// 	BeforeDeleteRevisionPodList := &corev1.PodList{}
	// 	if err := r.List(ctx, BeforeDeleteRevisionPodList); err != nil {
	// 		loggerSD.Error(err, err.Error())
	// 		break
	// 	}
	// 	count := 0
	// 	newPodDeploy := false
	// 	for _, pod := range BeforeDeleteRevisionPodList.Items {
	// 		if strings.HasPrefix(pod.Name, New_Revision_Number) && pod.Status.Phase == "Running" {
	// 			newPodDeploy = true // New Pod Not Ready, keep previous Revision alive
	// 			count += 1
	// 		} else if strings.HasPrefix(pod.Name, New_Revision_Number) && pod.Status.Phase != "Running" {
	// 			newPodDeploy = false
	// 		}
	// 	}
	// 	if count == 0 || !newPodDeploy {
	// 		loggerSD.Info("New Revision Pod NOT READY", "REV_NUMBER", New_Revision_Number)
	// 	} else { // only when New Revision Pod Ready, process to Delete Previous Revision Pods step
	// 		loggerSD.Info("New Revision Pod Running")
	// 		break
	// 	}
	// }

	// loggerSD.Info("Wait")
	// time.Sleep(5 * time.Second)

	// // New Revision Pods are READY now, Delete old Revision and old Revision pods
	// // Check if old Revision Pods are still Terminating. If YES delete old Revision, Then Delete pod
	// ReadyDeleteRevisionPodList := &corev1.PodList{}
	// if err := r.List(ctx, ReadyDeleteRevisionPodList); err != nil {
	// 	loggerSD.Error(err, err.Error())
	// } else {
	// 	count := 0 // count to ensure Delete Revision is only called one time in the PodList loop (when count = 1)
	// 	for _, pod := range ReadyDeleteRevisionPodList.Items {
	// 		if strings.HasPrefix(pod.Name, ServiceHouse_Current_Revision) {
	// 			targetpod := &corev1.Pod{
	// 				ObjectMeta: metav1.ObjectMeta{
	// 					Namespace: "default",
	// 					Name:      pod.Name,
	// 				},
	// 			}
	// 			count += 1
	// 			if count == 1 {
	// 				loggerSD.Info("Ask to delete Revision", "REVISION_NAME", ServiceHouse_Current_Revision)

	// 				err := serving.Revisions("default").Delete(context.Background(), ServiceHouse_Current_Revision, metav1.DeleteOptions{})
	// 				if err != nil {
	// 					loggerSD.Error(err, err.Error())
	// 				} else {
	// 					loggerSD.Info("Delete Revision ", "REVISION_NAME", ServiceHouse_Current_Revision)
	// 				}
	// 				time.Sleep(2 * time.Second)
	// 			}

	// 			if err := r.Delete(ctx, targetpod, client.GracePeriodSeconds(0)); err != nil {
	// 				loggerSD.Error(err, err.Error())
	// 			} else {
	// 				loggerSD.Info("Delete pod ", "POD_NAME", pod.Name)
	// 			}
	// 		}
	// 	}
	// }
	// }

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DRLScaleActionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&drlscalingv1.DRLScaleAction{}).
		Complete(r)
}

func CreateNewSVHouseConfiguration(con_value string, res_value string, podnumber_value string) *servingv1.Service {

	var NewServiceHouseConfiguration *servingv1.Service

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
							"autoscaling.knative.dev/target":        con_value,
							"autoscaling.knative.dev/initial-scale": podnumber_value,
							"autoscaling.knative.dev/min-scale":     podnumber_value,
						},
					},
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "deploy-a",
									Image: "vudinhdai2505/test-app:v5",
									Resources: corev1.ResourceRequirements{
										Requests: map[corev1.ResourceName]resource.Quantity{
											"memory": resource.MustParse("200Mi"),
										},
										Limits: map[corev1.ResourceName]resource.Quantity{
											"cpu": resource.MustParse(res_value),
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

	return NewServiceHouseConfiguration
}

func CreateNewSVSentiConfiguration(con_value string, res_value string, podnumber_value string) *servingv1.Service {

	var NewServiceSentiConfiguration *servingv1.Service

	NewServiceSentiConfiguration = &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentiment",
			Namespace: "default",
			Labels: map[string]string{
				"app": "sentiment",
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
							"app": "sentiment",
						},
						Annotations: map[string]string{
							"autoscaling.knative.dev/target":        con_value,
							"autoscaling.knative.dev/initial-scale": podnumber_value,
							"autoscaling.knative.dev/min-scale":     podnumber_value,
						},
					},
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "sentiment",
									Image: "mipearlska/sen_analysis_test:latest",
									Resources: corev1.ResourceRequirements{
										Limits: map[corev1.ResourceName]resource.Quantity{
											"cpu": resource.MustParse(res_value),
										},
									},
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 9980,
										},
									},
									ReadinessProbe: &corev1.Probe{
										PeriodSeconds:  13,
										TimeoutSeconds: 13,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return NewServiceSentiConfiguration
}

func CreateNewSVNumbrConfiguration(con_value string, res_value string, podnumber_value string) *servingv1.Service {

	var NewServiceNumbrConfiguration *servingv1.Service

	NewServiceNumbrConfiguration = &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "numberreg",
			Namespace: "default",
			Labels: map[string]string{
				"app": "numberreg",
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
							"app": "numberreg",
						},
						Annotations: map[string]string{
							"autoscaling.knative.dev/target":        con_value,
							"autoscaling.knative.dev/initial-scale": podnumber_value,
							"autoscaling.knative.dev/min-scale":     podnumber_value,
						},
					},
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "numberreg",
									Image: "ddocker122/number_recognization_service:v1",
									Resources: corev1.ResourceRequirements{
										Limits: map[corev1.ResourceName]resource.Quantity{
											"cpu": resource.MustParse(res_value),
										},
									},
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 5000,
										},
									},
									ReadinessProbe: &corev1.Probe{
										PeriodSeconds:  7,
										TimeoutSeconds: 7,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return NewServiceNumbrConfiguration
}

func ConvertResourceLimitToString(value resource.Quantity) string {
	temp1 := strings.Split(fmt.Sprintf("%v", value), "=")
	temp2 := strings.Split(temp1[0], " ")
	temp3 := temp2[0][2:]
	if len(temp3) == 1 {
		temp3 = temp3 + "000"
	}
	final := temp3 + "m"

	return final
}

func CalculateNewRevisionNumber(service_name string, current_revision string) string {
	// New Revision Number = current + 1 (from service-00009 to service-00010) (Below are string processing to get the new Revision ID/Number)
	tempstring := strings.Split(current_revision, "-")
	tempint, _ := strconv.Atoi(tempstring[len(tempstring)-1])
	rev_number := strconv.Itoa(tempint + 1)
	New_Revision_Number := service_name + "-" + strings.Repeat("0", 5-len(rev_number)) + rev_number

	return New_Revision_Number
}
