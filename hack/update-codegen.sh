#!/usr/bin/env bash

bash vendor/k8s.io/code-generator/generate-groups.sh  "applyconfiguration,client,deepcopy,informer,lister" \
github.com/leemingeer/noderesourcetopology/pkg/generated \
github.com/leemingeer/noderesourcetopology/pkg/apis topology:v1alpha1 --output-base ../../../ --go-header-file=./hack/boilerplate.go.txt