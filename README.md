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

To build the binary to run it use:

```
$ make build
$ ./svc-test
```

## Sketch

![diagram](https://raw.githubusercontent.com/K8sbykeshed/svc-tests/main/.diagram.png)
