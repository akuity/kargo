#!/bin/sh

set -x

helm uninstall kargo --namespace kargo

helm uninstall argo-rollouts --namespace argo-rollouts

helm uninstall argocd --namespace argocd

helm uninstall cert-manager --namespace cert-manager
