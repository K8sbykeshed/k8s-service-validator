FROM golang:1.16 as build
COPY . /k8s-service-validator
WORKDIR /k8s-service-validator
RUN go test -v -c -o svc-test ./tests

FROM docker:dind
COPY --from=build /k8s-service-validator/svc-test /svc-test
RUN apk add --update curl && apk add bash && rm -rf /var/cache/apk/*

# kubectl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /usr/local/bin

CMD ["./svc-test"]