# K8S Service Object Validator

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
