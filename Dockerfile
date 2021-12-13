
FROM golang:1.16 as build

COPY . /k8s-service-validator
WORKDIR /k8s-service-validator
RUN make build

FROM debian:buster-slim
COPY --from=build /k8s-service-validator/svc-test /

RUN mkdir $HOME/.kube
COPY ~/.kube/config $HOME/.kube
CMD ["bash", "-c", "/svc-test"]

