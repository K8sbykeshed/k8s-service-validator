FROM golang:1.17 as build
COPY . /k8s-service-validator
WORKDIR /k8s-service-validator
RUN go test -v -c -o svc-test ./tests

FROM debian:stretch-slim
COPY --from=build /k8s-service-validator/svc-test /svc-test
#RUN apk add --update curl && apk add bash && rm -rf /var/cache/apk/*

CMD ["./svc-test"]