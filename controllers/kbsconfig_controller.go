/*
Copyright Confidential Containers Contributors.

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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	confidentialcontainersorgv1alpha1 "github.com/confidential-containers/kbs-operator/api/v1alpha1"
	"github.com/go-logr/logr"
)

// KbsConfigReconciler reconciles a KbsConfig object
type KbsConfigReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	kbsConfig *confidentialcontainersorgv1alpha1.KbsConfig
	log       logr.Logger
	namespace string
}

//+kubebuilder:rbac:groups=confidentialcontainers.org,resources=kbsconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=confidentialcontainers.org,resources=kbsconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=confidentialcontainers.org,resources=kbsconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;update
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KbsConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *KbsConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx)
	_ = r.log.WithValues("kbsconfig", req.NamespacedName)
	r.log.Info("Reconciling KbsConfig")

	// Get the KbsConfig instance
	r.kbsConfig = &confidentialcontainersorgv1alpha1.KbsConfig{}
	err := r.Client.Get(ctx, req.NamespacedName, r.kbsConfig)
	// If the KbsConfig instance is not found, then just return
	// and do nothing
	if err != nil && errors.IsNotFound(err) {
		r.log.Info("KbsConfig not found")
		return ctrl.Result{}, nil
	}
	// If there is an error other than the KbsConfig instance not found,
	// then return with the error
	if err != nil {
		r.log.Error(err, "Failed to get KbsConfig")
		return ctrl.Result{}, err
	}

	// KbsConfig instance is found, so continue with rest of the processing

	// Check if the KbsConfig object is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isKbsConfigMarkedToBeDeleted := r.kbsConfig.GetDeletionTimestamp() != nil
	if isKbsConfigMarkedToBeDeleted {
		if contains(r.kbsConfig.GetFinalizers(), KbsFinalizerName) {
			// Run finalization logic for kbsFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			err := r.finalizeKbsConfig(ctx)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		// Remove kbsFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		r.log.Info("Removing kbsFinalizer")
		r.kbsConfig.SetFinalizers(remove(r.kbsConfig.GetFinalizers(), KbsFinalizerName))
		err := r.Update(ctx, r.kbsConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Create or update the KBS deployment
	err = r.deployOrUpdateKbsDeployment(ctx)
	if err != nil {
		r.log.Error(err, "Failed to create KBS deployment")
		return ctrl.Result{}, err
	}

	// Create or update the KBS service
	err = r.deployOrUpdateKbsService(ctx)
	if err != nil {
		r.log.Error(err, "Failed to create KBS service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// finalizeKbsConfig deletes the KBS deployment
func (r *KbsConfigReconciler) finalizeKbsConfig(ctx context.Context) error {
	// Delete the deployment
	r.log.Info("Deleting the KBS deployment")
	// Get the KbsDeploymentName deployment
	deployment := &appsv1.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: r.namespace,
		Name:      KbsDeploymentName,
	}, deployment)
	if err != nil {
		r.log.Error(err, "Failed to get KBS deployment")
		return err
	}
	err = r.Client.Delete(ctx, deployment)
	if err != nil {
		r.log.Error(err, "Failed to delete KBS deployment")
		return err
	}
	return nil
}

// deployOrUpdateKbsService returns a new service for the KBS instance
func (r *KbsConfigReconciler) deployOrUpdateKbsService(ctx context.Context) error {

	// Check if the service name kbs-service in r.namespace already exists
	// If it does, update the service
	// If it does not, create the service
	found := &corev1.Service{}

	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: r.namespace,
		Name:      KbsServiceName,
	}, found)

	if err != nil && errors.IsNotFound(err) {
		// Create the service
		r.log.Info("Creating a new service", "Service.Namespace", r.namespace, "Service.Name", KbsServiceName)
		service := r.newKbsService(ctx)
		// If service object is nil, return error
		if service == nil {
			r.log.Error(err, "Failed to get the KBS service definition")
			return err
		}
		err = r.Client.Create(ctx, service)
		if err != nil {
			r.log.Error(err, "Failed to create the KBS service")
			return err
		}
		// Service created successfully - return and requeue
		return nil
	} else if err != nil {
		r.log.Error(err, "Failed to get the KBS service")
		return err
	}

	// Service already exists, so update the service
	r.log.Info("Updating the service", "Service.Namespace", r.namespace, "Service.Name", KbsServiceName)
	service := r.newKbsService(ctx)
	// If service object is nil, return error
	if service == nil {
		r.log.Error(err, "Failed to get the KBS service definition")
		return err
	}
	err = r.Client.Update(ctx, service)
	if err != nil {
		r.log.Error(err, "Failed to update the KBS service")
		return err
	}
	// Service updated successfully - ret
	return nil
}

// newKbsService returns a new service for the KBS instance
func (r *KbsConfigReconciler) newKbsService(ctx context.Context) *corev1.Service {
	// Get the service type from the KbsConfig instance
	serviceType := r.kbsConfig.Spec.KbsServiceType
	// if the service type is not provided, default to ClusterIP
	if serviceType == "" {
		serviceType = corev1.ServiceTypeClusterIP
	}

	// Create a new service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.namespace,
			Name:      KbsServiceName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kbs",
			},
			Type: serviceType,
			Ports: []corev1.ServicePort{
				{
					Name:       "kbs-port",
					Protocol:   corev1.ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
	// Set KbsConfig instance as the owner and controller
	ctrl.SetControllerReference(r.kbsConfig, service, r.Scheme)
	return service
}

// deployOrUpdateKbsDeployment returns a new deployment for the KBS instance
func (r *KbsConfigReconciler) deployOrUpdateKbsDeployment(ctx context.Context) error {

	// Check if the deployment name kbs-deployment in r.namespace already exists
	// If it does, update the deployment
	// If it does not, create the deployment
	found := &appsv1.Deployment{}

	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: r.namespace,
		Name:      KbsDeploymentName,
	}, found)

	if err != nil && errors.IsNotFound(err) {
		// Create the deployment
		r.log.Info("Creating a new deployment", "Deployment.Namespace", r.namespace, "Deployment.Name", KbsDeploymentName)
		deployment := r.newKbsDeployment(ctx)
		// If deployment object is nil, return error
		if deployment == nil {
			r.log.Error(err, "Failed to create a deployment object", "Deployment.Namespace", r.namespace, "Deployment.Name", KbsDeploymentName)
			return fmt.Errorf("failed to create a deployment object")
		}
		err = r.Client.Create(ctx, deployment)
		if err != nil {
			// Failed to create the deployment
			r.log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", r.namespace, "Deployment.Name", KbsDeploymentName)
			return err
		} else {
			// Deployment created successfully
			r.log.Info("Created a new deployment", "Deployment.Namespace", r.namespace, "Deployment.Name", KbsDeploymentName)
			return nil
		}
	} else if err != nil {
		// Unknown error
		r.log.Error(err, "Failed to get Deployment")
		return err
	}
	// Update the found deployment and write the result back if there are any changes
	err = r.updateKbsDeployment(ctx, found)
	if err != nil {
		// Failed to update the deployment
		r.log.Error(err, "Failed to update Deployment", "Deployment.Namespace", r.namespace, "Deployment.Name", KbsDeploymentName)
		return err
	} else {
		// Deployment updated successfully
		r.log.Info("Updated Deployment", "Deployment.Namespace", r.namespace, "Deployment.Name", KbsDeploymentName)
	}

	// Add the kbsFinalizer to the KbsConfig if it doesn't already exist
	if !contains(r.kbsConfig.GetFinalizers(), KbsFinalizerName) {
		r.log.Info("Adding kbsFinalizer to KbsConfig")
		r.kbsConfig.SetFinalizers(append(r.kbsConfig.GetFinalizers(), KbsFinalizerName))
		err := r.Update(ctx, r.kbsConfig)
		if err != nil {
			r.log.Error(err, "Failed to update KbsConfig with kbsFinalizer")
			return err
		}
	}

	return nil

}

// newKbsDeployment returns a new deployment for the KBS instance
func (r *KbsConfigReconciler) newKbsDeployment(ctx context.Context) *appsv1.Deployment {
	// Set replica count
	replicas := int32(1)
	// Set rolling update strategy
	rollingUpdate := &appsv1.RollingUpdateDeployment{
		MaxUnavailable: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 1,
		},
	}
	// Set labels
	labels := map[string]string{
		"app": "kbs",
	}

	// Get Image Name from env variable if set
	imageName := os.Getenv("KBS_IMAGE_NAME")
	if imageName == "" {
		imageName = DefaultKbsImageName
	}

	// Create corev1.Volume array based on whether KbsConfigMapName, KbsAsConfigMapName, KbsAuthSecretName, and KbsRvpsConfigMapName are set
	// in the KbsConfig CR and whether the corresponding resources exist
	var volumes []corev1.Volume
	if r.kbsConfig.Spec.KbsConfigMapName != "" {
		// Check if the config map exists
		foundConfigMap := &corev1.ConfigMap{}
		err := r.Client.Get(ctx, client.ObjectKey{
			Namespace: r.namespace,
			Name:      r.kbsConfig.Spec.KbsConfigMapName,
		}, foundConfigMap)
		if err != nil && errors.IsNotFound(err) {
			// ConfigMap does not exist
			r.log.Error(err, "KbsConfigMapName does not exist", "ConfigMap.Namespace", r.namespace, "ConfigMap.Name", r.kbsConfig.Spec.KbsConfigMapName)
			return nil
		} else if err != nil {
			// Unknown error
			r.log.Error(err, "Failed to get KBS ConfigMap")
			return nil
		}
		// ConfigMap exists
		// Create the volume
		volumes = append(volumes, corev1.Volume{
			Name: "kbs-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: r.kbsConfig.Spec.KbsConfigMapName,
					},
				},
			},
		})
	}
	if r.kbsConfig.Spec.KbsRvpsConfigMapName != "" {
		// Check if the config map exists
		foundConfigMap := &corev1.ConfigMap{}
		err := r.Client.Get(ctx, client.ObjectKey{
			Namespace: r.namespace,
			Name:      r.kbsConfig.Spec.KbsRvpsConfigMapName,
		}, foundConfigMap)
		if err != nil && errors.IsNotFound(err) {
			// ConfigMap does not exist
			r.log.Error(err, "KbsRvpsConfigMapName does not exist", "ConfigMap.Namespace", r.namespace, "ConfigMap.Name", r.kbsConfig.Spec.KbsRvpsConfigMapName)
			return nil
		} else if err != nil {
			// Unknown error
			r.log.Error(err, "Failed to get KBS RVPS ConfigMap")
			return nil
		}
		// ConfigMap exists
		// Create the volume
		volumes = append(volumes, corev1.Volume{
			Name: "rvps-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: r.kbsConfig.Spec.KbsRvpsConfigMapName,
					},
				},
			},
		})
	}
	if r.kbsConfig.Spec.KbsAsConfigMapName != "" {
		// Check if the config map exists
		foundConfigMap := &corev1.ConfigMap{}
		err := r.Client.Get(ctx, client.ObjectKey{
			Namespace: r.namespace,
			Name:      r.kbsConfig.Spec.KbsAsConfigMapName,
		}, foundConfigMap)
		if err != nil && errors.IsNotFound(err) {
			// ConfigMap does not exist
			r.log.Error(err, "KbsAsConfigMapName does not exist", "ConfigMap.Namespace", r.namespace, "ConfigMap.Name", r.kbsConfig.Spec.KbsAsConfigMapName)
			return nil
		} else if err != nil {
			// Unknown error
			r.log.Error(err, "Failed to get KBS AS ConfigMap")
			return nil
		}
		// ConfigMap exists
		// Create the volume
		volumes = append(volumes, corev1.Volume{
			Name: "as-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: r.kbsConfig.Spec.KbsAsConfigMapName,
					},
				},
			},
		})
	}
	if r.kbsConfig.Spec.KbsAuthSecretName != "" {
		// Check if the secret exists
		foundSecret := &corev1.Secret{}
		err := r.Client.Get(ctx, client.ObjectKey{
			Namespace: r.namespace,
			Name:      r.kbsConfig.Spec.KbsAuthSecretName,
		}, foundSecret)
		if err != nil && errors.IsNotFound(err) {
			// Secret does not exist
			r.log.Error(err, "KbsAuthSecretName does not exist", "Secret.Namespace", r.namespace, "Secret.Name", r.kbsConfig.Spec.KbsAuthSecretName)
			return nil
		} else if err != nil {
			// Unknown error
			r.log.Error(err, "Failed to get KBS Auth Secret")
			return nil
		}
		// Secret exists
		// Create the volume
		volumes = append(volumes, corev1.Volume{
			Name: "auth-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: r.kbsConfig.Spec.KbsAuthSecretName,
				},
			},
		})
	}

	// Create corev1.VolumeMount array for the KBS container based on the volumes.
	// Use the same mountpath as the volume name
	volumeMounts := []corev1.VolumeMount{}
	for _, volume := range volumes {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volume.Name,
			MountPath: "/etc/" + volume.Name,
		})
	}

	// command array for the KBS container
	command := []string{
		"/usr/local/bin/kbs",
		"--socket",
		"0.0.0.0:8080",
		"--config",
		"/etc/kbs-config/kbs-config.json",
		"--auth-public-key",
		"/etc/auth-secret/kbs.pem",
		"--insecure-http",
	}

	// RunAsUser (root) 0
	runAsUser := int64(0)

	// Create the deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KbsDeploymentName,
			Namespace: r.namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				RollingUpdate: rollingUpdate,
				Type:          appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				// Add the KBS container
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kbs",
							Image: imageName,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
									Name:          "kbs",
								},
							},
							// Add command to start KBS
							Command: command,
							// Add SecurityContext
							SecurityContext: &corev1.SecurityContext{
								// Run as root user
								RunAsUser: &runAsUser,
							},
							// Add volume mount for KBS config
							VolumeMounts: volumeMounts,
						},
					},
					// Add volumes
					Volumes: volumes,
				},
			},
		},
	}
	return deployment
}

// updateKbsDeployment updates an existing deployment for the KBS instance
func (r *KbsConfigReconciler) updateKbsDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	err := r.Client.Update(ctx, deployment)
	if err != nil {
		// Failed to update the deployment
		r.log.Error(err, "Failed to update Deployment", "Deployment.Namespace", r.namespace, "Deployment.Name", "kbs-deployment")
		return err
	} else {
		// Deployment updated successfully
		r.log.Info("Updated Deployment", "Deployment.Namespace", r.namespace, "Deployment.Name", "kbs-deployment")
		return nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *KbsConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Get the namespace that the controller is running in
	r.namespace = os.Getenv("POD_NAMESPACE")
	if r.namespace == "" {
		r.namespace = KbsOperatorNamespace
	}

	// Create a new controller and add a watch for KbsConfig including the following secondary resources:
	// KbsConfigMap, KbsSecret, KbsAsConfigMap, KbsRvpsConfigMap in the same namespace as the controller
	return ctrl.NewControllerManagedBy(mgr).
		For(&confidentialcontainersorgv1alpha1.KbsConfig{}).
		// Watch for changes to ConfigMap, Secret that are in the same namespace as the controller
		// The ConfigMap and Secret are not owned by the KbsConfig
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(namespacePredicate(r.namespace)),
		).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(namespacePredicate(r.namespace)),
		).
		Complete(r)

}

// namespacePredicate is a custom predicate function that filters resources based on the namespace.
func namespacePredicate(namespace string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return isResourceInNamespace(e.Object, namespace)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isResourceInNamespace(e.ObjectNew, namespace)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return isResourceInNamespace(e.Object, namespace)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return isResourceInNamespace(e.Object, namespace)
		},
	}
}

// isResourceInNamespace checks if the resource is in the specified namespace.
func isResourceInNamespace(obj metav1.Object, namespace string) bool {
	return obj.GetNamespace() == namespace
}
