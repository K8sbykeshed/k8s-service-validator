# K8S Service-API Table Tests

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

## Build and run

To run it directly use:

```
$ make test
```

To build the binary and run it, use:

```
$ make build
$ ./svc-test
```

You must have a Kubernetes configuration at `$HOME/.kube/config`

### Running on a K8S cluster

## ClusterIP testing

```
reachability: correct:9, incorrect:0, result=true

51 <nil>
expected:

-		name-x/a	name-x/b	name-x/c
name-x/a	.		X		X	
name-x/b	.		X		X	
name-x/c	.		X		X	


97 <nil>
observed:

-		name-x/a	name-x/b	name-x/c
name-x/a	.		X		X	
name-x/b	.		X		X	
name-x/c	.		X		X	


97 <nil>
comparison:

-		name-x/a	name-x/b	name-x/c
name-x/a	.		.		.	
name-x/b	.		.		.	
name-x/c	.		.		.	
```

## Sketch

![diagram](https://raw.githubusercontent.com/K8sbykeshed/svc-tests/main/.diagram.png)
