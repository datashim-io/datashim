# Using cert-manager to rotate TLS certificates in Datashim

[cert-manager](https://cert-manager.io/) is a X.509 certificate controller for
Kubernetes and OpenShift workloads, capable of provisioning self-issued
certificates, setting up an in-house CA, and integrating with publicly available
CAs (e.g., letsencrypt).

!!! info 
    All the code samples below assume Datashim is installed in the `dlf` namespace.

## Installation

In this document we will not go through how to deploy cert-manager and instead
let the reader choose their preferred way to do so among the available ones
listed on [https://cert-manager.io/docs/installation/](https://cert-manager.io/docs/installation/).

## Requesting the Certificate

To get started with cert-manager, we will have to first create a namespaced
`Issuer` that will be able to issue us the certificate. We can simply `apply`
the following YAML to create a self-signed `Issuer`:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: datashim-issuer
  namespace: dlf
spec:
  selfSigned: {}
```

With this `Issuer` we are now able to provision a `Certificate` for the webhook
server by applying the following YAML:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: webhook-server-tls
  namespace: dlf
spec:
  secretName: webhook-server-tls
  secretTemplate:
    labels:
      app.kubernetes.io/name: dlf
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  isCA: false
  privateKey:
    algorithm: RSA
    encoding: PKCS1
    size: 2048
  usages:
    - server auth
    - client auth
  dnsNames:
    - webhook-server.dlf.svc
  issuerRef:
    name: datashim-issuer
    kind: Issuer
    group: cert-manager.io
```

This `Certificate` object will cause cert-manager to rotate the certificate 15
days before its expiration and provision a certificate valid 365 days.

To force cert-manager to provision the certificate we can manually delete the
associated secret with:

```commandline
kubectl delete secret -n dlf webhook-server-tls
```

## Restarting the webhook server

After provisioning the new certificate, we need to ensure the webhook server
picks up the new secret. We can do this by running:

```commandline
kubectl delete pod -n dlf -l name=dataset-operator
```

## Patching the MutatingWebhookConfiguration

As the `MutatingWebhookConfiguration` contains the CA used to create the
certificate, we need sync it with the one that cert-manager has used. Simply
run:

```commandline
CABUNDLE=$(kubectl get secret -n dlf webhook-server-tls -o jsonpath='{.data.ca\.crt}')
kubectl patch mutatingwebhookconfiguration -n dlf dlf-mutating-webhook-cfg --type='json' -p="[{'op': 'replace', 'path': '/webhooks/0/clientConfig/caBundle', 'value': \"$CABUNDLE\"}]"
```
