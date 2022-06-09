#!/usr/bin/env bash
set -e

go build -race -o cron ./cmd/kubeEP-cron && ./cron