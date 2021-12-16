# Demos

- Initial sig-network demo (Amim) https://www.youtube.com/watch?v=bFaym0zmHf8


# K8S Service-API Table Tests

[![GH Kube-Proxy - iptables - 1.21](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kube-proxy-iptables.yaml/badge.svg)](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kube-proxy-iptables.yaml)
[![GH Kube-Proxy - IPTables - 1.22](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kube-proxy-iptables-122.yaml/badge.svg)](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kube-proxy-iptables-122.yaml)
[![GH Kube-Proxy - ipvs](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kube-proxy-ipvs.yaml/badge.svg)](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kube-proxy-ipvs.yaml)
[![GH KPNG - nftables](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kpng-nftables.yaml/badge.svg)](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kpng-nftables.yaml)
[![GH KPNG - ipvs](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kpng-ipvs.yaml/badge.svg)](https://github.com/K8sbykeshed/k8s-service-validator/actions/workflows/kpng-ipvs.yaml)

## Problem

The current upstream K8s sig-network tests are difficult to interpret in terms of failures, and
schedule pods randomly to nodes, while fixing endpoints to the 1st node in a list.

Although we cant deprecate these tests - b/c of their inertia, we need a better tool to diagnose
Service / kube-proxy implementations across providers.  Examples of such providers are:

- KPNG (https://github.com/kubernetes-sigs/kpng/)
- AntreaProxy (https://github.com/vmware-tanzu/antrea/pull/772)
- Cilliums Kube-proxy implementation (https://docs.cilium.io/en/v1.9/gettingstarted/kubeproxy-free/)
- Calico's proxying options (https://thenewstack.io/beyond-kube-proxy-tigera-calico-harnesses-ebpf-for-a-faster-data-plane/)
- ... others ? feel free to PR / add here ! ...

As these implementations of kube proxy diverge over time, knowing wether loadbalancing is failing due to the source
or target of traffic, wether all node proxies or just a few are broken, and wether configurations like node-local
endpoints, or terminating endpoint scenarios are causing specific issues, becomes increasingly important for comparing services.

## Solution

To solve the problem of how we can reason about different services, on different providers, with a low cognitive overhead, we'll 
Visualize service availability in a k8s cluster using... Tables! 

With table tests we can rapidly compare what may or may not be broken and generate hypothesis, as is done in the network policy tests in upstream k8s, https://kubernetes.io/blog/2021/04/20/defining-networkpolicy-conformance-cni-providers/.

As an example, the below test probes 3 endpoints, from 3 endpoints.  Assuming uniform spreading of services, this can detect potential problems in service loadbalancing that are specific to nodes sources / targets... 

```
-		name-x/a	name-x/b	name-x/c
name-x/a	.		X		X	
name-x/b	.		X		X	
name-x/c	.		X		X	
```

In this case, we can see that pods "b" and "c" in namespace x are not reachable from ANY pod, meaning that theres a problem
with any kube-proxy , on any node, i.e. the loadbalancing is fundamentally not working (the `Xs` are failures).

This suite of tests validates objects rules and various scenarios using
Kubernetes services as a way of control tests heuristics, as for now the 
following tests are available:

- ClusterIP
- ExternalName
- Loadbalancer
- NodePort

# Details/Contributing

This is just an initial experimental repo but we'd like to fully implement this as a KEP and add test coverage to upstream K8s, 
if there is consensus in the sig-network group for this.

## Build and run - development

Create a local cluster using the Kind, make sure your cluster have 1+ nodes,
install MetalLB:

```
$ kind create cluster --config=hack/kind-multi-worker.yaml
$ hack/install_metallb.sh
```

To run the tests directly you can use:

```
$ make test
```

To build the binary and run it, use:

```
$ make build
$ ./svc-test
```

### Using E2E tests

Download the Kubernetes repository and build the tests binary

```
$ make WHAT=test/e2e/e2e.test
+++ [1103 15:29:10] Building go targets for linux/amd64:
    ./vendor/k8s.io/code-generator/cmd/prerelease-lifecycle-gen
> non-static build: k8s.io/kubernetes/./vendor/k8s.io/code-generator/cmd/prerelease-lifecycle-gen
Generating prerelease lifecycle code for 28 targets
+++ [1103 15:29:13] Building go targets for linux/amd64:
    ./vendor/k8s.io/code-generator/cmd/deepcopy-gen
> non-static build: k8s.io/kubernetes/./vendor/k8s.io/code-generator/cmd/deepcopy-gen
Generating deepcopy code for 236 targets
+++ [1103 15:29:19] Building go targets for linux/amd64:
    ./vendor/k8s.io/code-generator/cmd/defaulter-gen
> non-static build: k8s.io/kubernetes/./vendor/k8s.io/code-generator/cmd/defaulter-gen
Generating defaulter code for 94 targets
+++ [1103 15:29:26] Building go targets for linux/amd64:
    ./vendor/k8s.io/code-generator/cmd/conversion-gen
> non-static build: k8s.io/kubernetes/./vendor/k8s.io/code-generator/cmd/conversion-gen
Generating conversion code for 130 targets
+++ [1103 15:29:38] Building go targets for linux/amd64:
    ./vendor/k8s.io/kube-openapi/cmd/openapi-gen
> non-static build: k8s.io/kubernetes/./vendor/k8s.io/kube-openapi/cmd/openapi-gen
Generating openapi code for KUBE
Generating openapi code for AGGREGATOR
Generating openapi code for APIEXTENSIONS
Generating openapi code for CODEGEN
Generating openapi code for SAMPLEAPISERVER
+++ [1103 15:29:47] Building go targets for linux/amd64:
    test/e2e
> non-static build: k8s.io/kubernetes/test/e2e
```

There will be a new compiled binary under `_ouput/bin/e2e.test` with around 150Mb, you
can use it to run your sig-network tests as:

```
_output/bin/e2e.test -ginkgo.focus="\[sig-network\]" \
    -ginkgo.skip="\[Feature:(Networking-IPv6|Example|Federation|PerformanceDNS)\]|LB.health.check|LoadBalancer|load.balancer|GCE|NetworkPolicy|DualStack" \
    --provider=local \ 
    --kubeconfig=.kube/config
```

You must have a Kubernetes configuration at `$HOME/.kube/config`

### Running on a K8S cluster

This test requires mode than 1 Nodes, and it guarantees the proper spread of pods across the existent
nodes creating len(nodes) pods. The examples above have 4 nodes (3 workers + 1 master).

## ClusterIP testing

On this example we have a `pod-1`, ` pod-2`, `pod-3` and `pod-4`, the first lines of the probe.go in the logging
shows `pod-1` probing the other probes on port `80`, this probe is repeat across all other pods, the reachability
matrix shows the result of the connections outcomes.

```
{"level":"info","ts":1621793385.240396,"caller":"manager/helper.go:45","msg":"Validating reachability matrix... (== FIRST TRY ==)"}
{"level":"info","ts":1621793385.3714652,"caller":"manager/probe.go:114","msg":"kubectl exec pod-1 -c cont-80-tcp -n x-12348 -- /agnhost connect s-x-12348-pod-3.x-12348.svc.cluster.local:80 --timeout=5s --protocol=tcp"}
{"level":"info","ts":1621793385.3720098,"caller":"manager/probe.go:114","msg":"kubectl exec pod-1 -c cont-80-tcp -n x-12348 -- /agnhost connect s-x-12348-pod-2.x-12348.svc.cluster.local:80 --timeout=5s --protocol=tcp"}
{"level":"info","ts":1621793385.385194,"caller":"manager/probe.go:114","msg":"kubectl exec pod-1 -c cont-80-tcp -n x-12348 -- /agnhost connect s-x-12348-pod-1.x-12348.svc.cluster.local:80 --timeout=5s --protocol=tcp"}
{"level":"info","ts":1621793385.4150555,"caller":"manager/probe.go:114","msg":"kubectl exec pod-1 -c cont-80-tcp -n x-12348 -- /agnhost connect s-x-12348-pod-4.x-12348.svc.cluster.local:80 --timeout=5s --protocol=tcp"}
...
// repreat for all pods

reachability: correct:16, incorrect:0, result=true

52 <nil>
expected:

-               x-12348/pod-1   x-12348/pod-2   x-12348/pod-3   x-12348/pod-4
x-12348/pod-1   .               .               .               .
x-12348/pod-2   .               .               .               .
x-12348/pod-3   .               .               .               .
x-12348/pod-4   .               .               .               .


176 <nil>
observed:

-               x-12348/pod-1   x-12348/pod-2   x-12348/pod-3   x-12348/pod-4
x-12348/pod-1   .               .               .               .
x-12348/pod-2   .               .               .               .
x-12348/pod-3   .               .               .               .
x-12348/pod-4   .               .               .               .


176 <nil>
comparison:

-               x-12348/pod-1   x-12348/pod-2   x-12348/pod-3   x-12348/pod-4
x-12348/pod-1   .               .               .               .
x-12348/pod-2   .               .               .               .
x-12348/pod-3   .               .               .               .
x-12348/pod-4   .               .               .               .
```

## Sketch

![diagram](https://raw.githubusercontent.com/K8sbykeshed/svc-tests/main/.diagram.png)


# Plan
- Initial demo at sig-network (done), establishing agreement on future of service tests.
- Establish parity w/ existing sig-network tests
- Use this repo to do all the validation for KPNG
- demo KPNG with the new service lb tests at sig-network
- establish a kubernetes-sigs/... repo for this framework, to standardize the CI for KPNG 
- use these tests as a new test-infra job in sig-network for in-tree kube proxy
- eventually remove old service lb tests from sig-network if community agrees
