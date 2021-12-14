#!/usr/bin/env bash

exec ./kind create cluster --config=hack/kind-multi-worker.yaml
exec ./svc-test