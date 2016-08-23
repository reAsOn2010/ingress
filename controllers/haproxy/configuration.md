## Contents
* [Customizing Haproxy](#customizing-haproxy)
* [Custom NGINX template](#custom-nginx-template)
* [Annotations](#annotations)
* [Rewrite](#rewrite)

### Customizing Haproxy
there are 2 ways to customize haproxy

1. annoations: [annotate the ingress](#annotations), use this if you want a specific configuration for the site defined in the ingress rule
2. custom template

#### Annotations

|Name                 |type|
|---------------------------|------|
|[ingress.kubernetes.io/rewrite-target](#rewrite)|URI|

#### Custom NGINX template

The Haproxy template is located in the file `/nginx.tmpl`. Mounting a volume is possible to use a custom version.
Use the [custom-template](examples/custom-template/README.md) example as a guide

**Please note the template is tied to the go code. Be sure to no change names in the variable `$cfg`**

### Rewrite

In some scenarios the exposed URL in the backend service differs from the specified path in the Ingress rule. Without a rewrite any request will return 404.
Set the annotation `ingress.kubernetes.io/rewrite-target` to the path expected by the service.

Please check the [rewrite](examples/rewrite/README.md) example
