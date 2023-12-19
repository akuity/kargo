#!/bin/sh

set -x

helm uninstall kargo --namespace kargo

helm uninstall rollouts --namespace rollouts

helm uninstall argocd --namespace argocd

helm uninstall cert-manager --namespace cert-manager
