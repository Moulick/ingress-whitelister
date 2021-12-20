# Ingress Whitelister

## What is Ingress Whitelister?

Ingress Whitelister adds annotations to your ingress objects based on labels. It is a very simple operator whose current
sole purpose is to compile a list of ip addresses and add it as an given annotation

This operator is built using [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).

## Input

The operator takes `IPWhitelistConfig` as input. For every ingress resource, it will check the label and compile the set
of IP addresses which should be whitelisted for the ingress

## Installation

`make install` will generate and apply the CRDs required to your cluster

`make deploy` will generate and deploy the operator to your cluster

Or take a look at the [Makefile](Makefile) for more advances use cases

The docker image can be found on dockerhub [moulick/ingress-whitelister](https://hub.docker.com/r/moulick/ingress-whitelister)

## Examples

A fully defined sample of `IPWhitelistConfig` is given in the [config/samples](config/samples)

## Considerations

1. Duplicate labels can cause hot looping and cause flip flopping in the annotations.
2. Currently the operator reconciles only on ingress object

## TODO

1. Add a webhook to validate IP addresses
2. Add reconciler on `IPWhitelistConfig` as well
3. Add a way to handle duplicate labels
