#!/bin/sh

helm uninstall kargo --namespace kargo

helm uninstall argocd --namespace argocd

helm uninstall cert-manager --namespace cert-manager
