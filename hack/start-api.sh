#!/bin/bash
# password hash is 'admin'

ADMIN_ACCOUNT_ENABLED=true \
ADMIN_ACCOUNT_TOKEN_AUDIENCE=kargo.demo.akuity.io \
ADMIN_ACCOUNT_TOKEN_ISSUER=http://kargo.demo.akuity.io \
ADMIN_ACCOUNT_TOKEN_TTL=1h \
ADMIN_ACCOUNT_PASSWORD_HASH='$2b$15$QYevw0zQuU9bMY3UofSsvuekk7yAMPc9akuWEWf6CLfRgFWJpdIkS' \
ADMIN_ACCOUNT_TOKEN_SIGNING_KEY='aXdpc2h0b3dhc2hteWlyaXNod3Jpc3R3YXRjaA==' \
KUBECONFIG=~/.kube/config \
ARGOCD_KUBECONFIG=~/.kube/config \
    go run ./cmd/controlplane api