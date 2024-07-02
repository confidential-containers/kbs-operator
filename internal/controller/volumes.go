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

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KbsConfigReconciler) createEmptyDirVolume(volumeName string) (*corev1.Volume, error) {
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	}
	return &volume, nil
}

func (r *KbsConfigReconciler) createSecretVolume(ctx context.Context, volumeName string, secretName string) (*corev1.Volume, error) {
	if secretName == "" {
		return nil, fmt.Errorf("Secret name hasn't been provided for volume " + volumeName)
	}

	r.log.Info("Retrieving details for ", "Secret.Name", secretName, "Secret.Namespace", r.namespace)
	foundSecret := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: r.namespace,
		Name:      secretName,
	}, foundSecret)
	if err != nil {
		return nil, err
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
	return &volume, nil
}

// Method to add KbsSecretResources to the KBS volumes
func (r *KbsConfigReconciler) createKbsSecretResourcesVolume(ctx context.Context) ([]corev1.Volume, error) {
	var secretVolumes []corev1.Volume
	if r.kbsConfig.Spec.KbsSecretResources != nil {
		for _, secretResource := range r.kbsConfig.Spec.KbsSecretResources {
			r.log.Info("Retrieving KbsSecretResource", "Secret.Namespace", r.namespace, "Secret.Name", secretResource)
			foundSecret := &corev1.Secret{}
			err := r.Client.Get(ctx, client.ObjectKey{
				Namespace: r.namespace,
				Name:      secretResource,
			}, foundSecret)
			if err != nil {
				return nil, err
			}

			secretVolumes = append(secretVolumes, corev1.Volume{
				Name: secretResource,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secretResource,
					},
				},
			})
		}
	}
	return secretVolumes, nil
}

func (r *KbsConfigReconciler) createConfigMapVolume(ctx context.Context, volumeName string, configMapName string) (*corev1.Volume, error) {
	if configMapName == "" {
		return nil, fmt.Errorf("ConfigMap name hasn't been provided for volume " + volumeName)
	}

	r.log.Info("Retrieving details for ", "ConfigMap.Name", configMapName, "ConfigMap.Namespace", r.namespace)
	foundConfigMap := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: r.namespace,
		Name:      configMapName,
	}, foundConfigMap)
	if err != nil {
		return nil, err
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
	return &volume, nil
}

func createVolumeMount(volumeName string, mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
	}
}

func createVolumeMountWithSubpath(volumeName string, mountPath string, subPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		SubPath:   subPath,
	}
}
