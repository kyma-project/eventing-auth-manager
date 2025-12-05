[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/eventing-auth-manager)](https://api.reuse.software/info/github.com/kyma-project/eventing-auth-manager)

# Eventing Auth Manager

Eventing Auth Manager is a central component that is deployed in Kyma Control Plane (KCP). The component manages applications in the SAP Cloud Identity Services - Identity Authentication by creating and deleting them based on the creation or deletion of a managed Kyma runtime.

For more information, see the [`/contributor`](./docs/contributor) folder.

## Getting Started

### Prerequisites
 
You must have a Kubernetes cluster. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing purposes, or use a remote one.

> ### Note 
> Your controller automatically uses the current context in your kubeconfig file. To check your current context, run `kubectl cluster-info`.

### Preparing the Clusters for Local Development

To run the controller locally, you need to have two clusters available. One cluster is used to run the controller, and the other cluster is used as a target for the created Secret.

#### Prepare the Cluster that Is Used to Run the Controller

1. Create the namespace to deploy the manager and the SAP Cloud Identity Services - Identity Authentication credential Secret:

   ```sh
   kubectl create ns kcp-system
   ```

2. Create the Secret for the SAP Cloud Identity Services - Identity Authentication credentials specified by: 

- `TEST_EVENTING_AUTH_IAS_USER`
- `TEST_EVENTING_AUTH_IAS_PASSWORD`
- `TEST_EVENTING_AUTH_IAS_URL`

   ```sh
   kubectl create secret generic eventing-auth-ias-creds -n kcp-system --from-literal=username=$TEST_EVENTING_AUTH_IAS_USER --from-literal=password=$TEST_EVENTING_AUTH_IAS_PASSWORD --from-literal=url=$TEST_EVENTING_AUTH_IAS_URL
   ```

3. Create the Secret containing the kubeconfig of the cluster on which the `eventing-webhook-auth` Secret is created by specifying `PATH_TO_TARGET_CLUSTER_KUBECONFIG` and `KYMA_CR_NAME`.

   ```sh
   kubectl create secret generic kubeconfig-$KYMA_CR_NAME -n kcp-system --from-file=config=$PATH_TO_TARGET_CLUSTER_KUBECONFIG
   ```

#### Prepare the Target Cluster

Create the namespace in which the `eventing-webhook-auth` Secret is created in the target cluster:

```sh
kubectl create ns kyma-system
```

### Running in the Cluster

1. Install the Kyma and EventingAuth Custom Resource Definitions (CRDs):

   ```sh
   make install
   ```

2. Update the name of the custom resource (CR) in `config/samples/operator_v1beta2_kyma.yaml` to contain the name of the kubeconfig secret created in [Preparing the clusters](#preparing-the-clusters). The Kyma CR name is the same as the target Kubernetes cluster name.

3. Install the CRs instances:

   ```sh
   kubectl apply -f config/samples/
   ```

4. Build and push your image to the location specified by `IMG`:

   ```sh
   make docker-build docker-push IMG=<some-registry>/eventing-auth-manager:tag
   ```

5. Deploy the controller to the cluster with the image specified by `IMG`:

   ```sh
   make deploy IMG=<some-registry>/eventing-auth-manager:tag
   ```

### Uninstall CRDs

To delete the CRDs from the cluster, run:

```sh
make uninstall
```

### Undeploy controller

To undeploy the controller from the cluster, run:

```sh
make undeploy
```

### Configuring Integration Tests

To execute the tests, run:

```sh
make test
```

#### SAP Cloud Identity Services - Identity Authentication Stub

By default, the integration tests use a stub for the SAP Cloud Identity Services - Identity Authentication API. This stub returns. It's also possible to use the real SAP Cloud Identity Services - Identity Authentication API by setting all the following environment variables:

```sh
export TEST_EVENTING_AUTH_IAS_URL=https://my-tenant.accounts.ondemand.com
export TEST_EVENTING_AUTH_IAS_USER=user@sap.com
export TEST_EVENTING_AUTH_IAS_PASSWORD=password
```

#### Target Cluster

By default, the integration tests use a local control plane created by [envtest](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest).

It's also possible to use a real target cluster by setting the following environment variable:

```sh
# The path to the kubeconfig of the cluster
export TEST_EVENTING_AUTH_TARGET_KUBECONFIG_PATH=/some/path/.kube/config
```

## Contributing

This project aims to follow the [Kubernetes Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/), which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

For general contributing guidelines, see the [Contributing Rules](CONTRIBUTING.md).

### Test it Out

1. Install the CRDs into the cluster:

   ```sh
   make install
   ```

2. In a new terminal window, run your controller:

   ```sh
   make run
   ```

> ### Tip
> You can also trigger both operations by running: `make install run`.

### Modify the API definitions

If you are editing the API definitions, generate the manifests, such as CRs or CRDs, using the following command:

```sh
make manifests
```

> ### Note 
> Run `make --help` for more information on all potential `make` targets.

For more information, see [the official Kubebuilder documentation](https://book.kubebuilder.io/introduction.html).

## Code of Conduct

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing

See the [license](./LICENSE) file.
