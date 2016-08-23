# Haproxy Ingress Controller

This is a Haproxy Ingress controller. See [Ingress controller documentation](https://github.com/kubernetes/contrib/blob/master/ingress/controllers/README.md) for details on how it works.

## Content
* [Build](#build)
* [Conventions](#conventions)
* [Dry running](#dry-running-the-ingress-controller)
* [HTTPS](#https)
* [Deployment](#deployment)
* [Haproxy customization](configuration.md)

## Build

The make file will compile the binary, build image and push the image to remote registry, the image will be taged as ${IMAGE_NAME}:${IMAGE_TAG}
```
$ IMAGE_NAME=xxx IMAGE_TAG=0.0 make
```

## Conventions

Anytime we reference a tls secret, we mean (x509, pem encoded, RSA 2048, etc). You can generate such a certificate with: 
`openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout $(KEY) -out $(CERT) -subj "/CN=$(HOST)/O=$(HOST)"`
and creat the secret via `kubectl create secret tls --key file --cert file`

## Dry running the Ingress controller

```
$ make local
$ bin/controller --running-in-cluster=false
```

## HTTPS

Create ssl cert and key pair and create kubernetes secret:
```
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout $(KEY) -out $(CERT) -subj "/CN=$(HOST)/O=$(HOST)"
kubectl create secret tls $(SECRET) --key file --cert file
```

Referencing this secret in an Ingress will tell the Ingress controller to secure the channel from the client to the loadbalancer using TLS:
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: no-rules-map
spec:
  tls:
    - hosts:
      - $(HOST)
      secretName: $(SECRET)
  backend:
    serviceName: s1
    servicePort: 80
```

## Deployment
Loadbalancers now are created via pods (fill the missing variables for docker images and pin host)
```
$ kubectl create -f examples/default/pods.yaml
```

### Debug & Troubleshooting

Using the flag `--v=XX` it is possible to increase the level of logging.
In particular:
- `--v=2` shows details using `diff` about the changes in the configuration in nginx
- `--v=3` shows details about the service, Ingress rule, endpoint changes and it dumps the haproxy configuration in JSON format
