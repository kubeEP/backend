#!/usr/bin/env bash
set -e

go build -race -o app ./cmd/kubeEP && ./app