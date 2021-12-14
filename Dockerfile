FROM golang:1.16 as build
COPY . /k8s-service-validator
WORKDIR /k8s-service-validator
RUN go test -v -c -o svc-test ./tests

FROM docker:dind
COPY --from=build /k8s-service-validator/svc-test /svc-test
COPY --from=build /k8s-service-validator/hack /hack

#kind
RUN apk add --update curl && apk add bash && rm -rf /var/cache/apk/*
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.11.1/kind-linux-amd64
RUN chmod +x ./kind && \
    chmod +x /hack/entrypoint.sh && \
    chmod +x /usr/local/bin/docker-entrypoint.sh

CMD ["hack/entrypoint.sh"]

