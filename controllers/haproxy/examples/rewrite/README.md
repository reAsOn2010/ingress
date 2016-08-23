Create an Ingress rule with a rewrite annotation:
```
$ echo "
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    ingress.kubernetes.io/rewrite-target: /
  name: rewrite
  namespace: default
spec:
  rules:
  - host: rewrite.bar.com
    http:
      paths:
      - backend:
          serviceName: echoheaders
          servicePort: 80
        path: /something/
" | kubectl create -f -
```
