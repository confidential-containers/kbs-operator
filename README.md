# trustee-operator

The `trustee-operator` manages the lifecycle of [trustee](https://github.com/confidential-containers/trustee) along with it's configuration when deployed
in a Kubernetes cluster

## Description

The operator manages a Kubernetes custom resource named: `KbsConfig`. Following are the key fields of the
`KbsConfig` custom resource definition

```golang
// KbsConfigSpec defines the desired state of KbsConfig
type KbsConfigSpec struct {
  // KbsConfigMapName is the name of the configmap that contains the KBS configuration
  KbsConfigMapName string `json:"kbsConfigMapName,omitempty"`

  // KbsAsConfigMapName is the name of the configmap that contains the KBS AS configuration
  // Required only when MicroservicesDeployment is set
  // +optional
  KbsAsConfigMapName string `json:"kbsAsConfigMapName,omitempty"`

  // KbsRvpsConfigMapName is the name of the configmap that contains the KBS RVPS configuration
  // Required only when MicroservicesDeployment is set
  // +optional
  KbsRvpsConfigMapName string `json:"kbsRvpsConfigMapName,omitempty"`

  // kbsRvpsRefValuesConfigMapName is the name of the configmap that contains the RVPS reference values
  KbsRvpsRefValuesConfigMapName string `json:"kbsRvpsRefValuesConfigMapName,omitempty"`

  // KbsAuthSecretName is the name of the secret that contains the KBS auth secret
  KbsAuthSecretName string `json:"kbsAuthSecretName,omitempty"`

  // KbsServiceType is the type of service to create for KBS
  // Default value is ClusterIP
  // +optional
  KbsServiceType corev1.ServiceType `json:"kbsServiceType,omitempty"`

  // KbsDeploymentType is the type of KBS deployment
  // It can assume one of the following values:
  //    AllInOneDeployment: all the KBS components will be deployed in the same container
  //    MicroservicesDeployment: all the KBS components will be deployed in separate containers
  // +kubebuilder:validation:Enum=AllInOneDeployment;MicroservicesDeployment
  // Default value is AllInOneDeployment
  // +optional
  KbsDeploymentType DeploymentType `json:"kbsDeploymentType,omitempty"`

  // KbsHttpsKeySecretName is the name of the secret that contains the KBS https private key
  KbsHttpsKeySecretName string `json:"kbsHttpsKeySecretName,omitempty"`

  // KbsHttpsCertSecretName is the name of the secret that contains the KBS https certificate
  KbsHttpsCertSecretName string `json:"kbsHttpsCertSecretName,omitempty"`

  // KbsSecretResources is an array of secret names that contain the keys required by clients
  // +optional
  KbsSecretResources []string `json:"kbsSecretResources,omitempty"`

  // kbsResourcePolicyConfigMapName is the name of the configmap that contains the Resource Policy
  // +optional
  KbsResourcePolicyConfigMapName string `json:"kbsResourcePolicyConfigMapName,omitempty"`

  // tdxConfigSpec is the struct that hosts the TDX specific configuration
  // +optional
  TdxConfigSpec TdxConfigSpec `json:"tdxConfigSpec,omitempty"`
}

// TdxConfigSpec defines the desired state for TDX configuration
type TdxConfigSpec struct {
  // kbsTdxConfigMapName is the name of the configmap containing sgx_default_qcnl.conf file
  // +optional
  KbsTdxConfigMapName string `json:"kbsTdxConfigMapName,omitempty"`
}```

Note: the default deployment type is ```MicroservicesDeployment```.
The examples below apply to this mode.

An example configmap for the KBS configuration looks like this:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kbs-config-grpc
  namespace: kbs-operator-system
data:
  kbs-config.json: |
    {
        "insecure_http" : false,
        "sockets": ["0.0.0.0:8080"],
        "auth_public_key": "/etc/auth-secret/kbs.pem",
        "private_key": "/etc/https-key/key.pem",
        "certificate": "/etc/https-cert/cert.pem",
        "attestation_token_config": {
          "attestation_token_type": "CoCo"
        },
        "grpc_config" : {
          "as_addr": "http://127.0.0.1:50004"
        }
    }
```

If HTTPS support is not needed, please set `insecure_http=true` and no need to specify the attributes `private_key` and `certificate`.

An example configmap for AS config looks like this:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: as-config-grpc
  namespace: kbs-operator-system
data:
  as-config.json: |
    {
        "work_dir": "/opt/confidential-containers/attestation-service",
        "policy_engine": "opa",
        "rvps_config": {
           "remote_addr":"http://127.0.0.1:50003"
        },
        "attestation_token_broker": "Simple",
        "attestation_token_config": {
          "duration_min": 5
        }
    }
```

Currently these configmaps needs to be created during deployment.
In subsequent releases we'll look into having these configmaps created by the operator based on user inputs.

A sample `KbsConfig` custom resource

```yaml
apiVersion: confidentialcontainers.org/v1alpha1
kind: KbsConfig
metadata:  
  name: kbsconfig-sample
  namespace: kbs-operator-system
spec:
  # KBS configuration
  kbsConfigMapName: kbs-config
  # AS configuration
  kbsAsConfigMapName: as-config  
  # RVPS configuration
  kbsRvpsConfigMapName: rvps-config-grpc
  # reference values config map
  kbsRvpsReferenceValuesMapName: rvps-reference-values
  # authentication secret
  kbsAuthSecretName: kbs-auth-public-key
  # service type
  kbsServiceType: ClusterIP
  # deployment type
  kbsDeploymentType: MicroservicesDeployment
  # HTTPS support
  kbsHttpsKeySecretName: kbs-https-key
  kbsHttpsCertSecretName: kbs-https-certificate
  # K8s Secrets to be made available to KBS clients
  kbsSecretResources: ["kbsres1"]
  # Resource policy
  kbsResourcePolicyConfigMapName: resource-policy
  # TDX configuration file
  kbsTdxConfigMapName: tdx-config
```

## Getting Started

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster

- Export env variables.

  Set `REGISTRY` environment variable to point to your container registry.
  For example:

  ```sh
  export REGISTRY=quay.io/user
  ```

- Build and push your image to the location specified by `IMG`.

  ```sh
  make docker-build docker-push IMG=${REGISTRY}/kbs-operator:latest
  ```

  Change the tag from `latest` to any other based on your requirements.
  Also ensure that the image is public.

- Deploy the controller to the cluster with the image specified by `IMG`.

  ```sh
  make deploy IMG=${REGISTRY}/kbs-operator:latest
  ```

- Deployment of CRDs, ConfigMaps and Secrets

  This is an example. Change it to real values as per your requirements.

  It is recommended to uncomment the secret generation for the trustee authorization in the  [kustomization.yaml](config/samples/microservices/kustomization.yaml), for both public and private key (`kbs-auth-public-key` and `kbs-client` secrets)

  ```sh
  cd config/samples/microservices
  # or config/samples/all-in-one for the integrated mode

  # create authentication keys
  openssl genpkey -algorithm ed25519 > privateKey
  openssl pkey -in privateKey -pubout -out kbs.pem
  
  # create all the needed resources
  kubectl apply -k .
  ```

  Among various things, the command above is also responsible for injecting reference values into the RVPS component. The default json file is an empty sequence, but you may want to inject real values by applying a ConfigMap like the one below:

  ``` yaml
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: rvps-reference-values
    namespace: kbs-operator-system
  data:
    reference-values.json: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: rvps-reference-values
        namespace: kbs-operator-system
      data:
        reference-values.json: |
          [
            {
              "name": "sample.svn",
              "expired": "2025-01-01T00:00:00Z",
              "hash-value": [
                {
                  "alg": "sha256",
                  "value": "1"
                }
              ]
            }
          ]
  ```

  It is also possible to create the K8s secrets (a commented out example is provided in the [kustomization.yaml](config/samples/microservices/kustomization.yaml)). To enable the secrets you'd need to uncomment the relevant secret generator entry and patch.
  
### Uninstall CRDs

To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller

UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing

Contributions are most welcome. Please take a look at the [guide](https://github.com/confidential-containers/confidential-containers/blob/main/CONTRIBUTING.md) for more details.

### How it works

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out

- Install the CRDs into the cluster.

  ```sh
  make install
  ```

- Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

  ```sh
  make run
  ```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright Confidential Containers Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
