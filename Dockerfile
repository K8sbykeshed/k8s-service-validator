FROM golang:1.17 as build
COPY . /k8s-service-validator
WORKDIR /k8s-service-validator
RUN go test -v -c -o svc-test ./tests

FROM debian:stretch-slim
COPY --from=build /k8s-service-validator/svc-test /svc-test
COPY --from=build /k8s-service-validator/run.sh /run.sh

RUN chmod +x ./run.sh
ENTRYPOINT ["./run.sh"]
