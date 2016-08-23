package rewrite

import (
	"errors"

	"k8s.io/kubernetes/pkg/apis/extensions"
)

const (
	rewriteTo = "ingress.kubernetes.io/rewrite-target"
	slash     = "/"
)

type Rewrite struct {
	Origin string
	Target string
}

type ingAnnotations map[string]string

func (a ingAnnotations) rewriteTo() string {
	val, ok := a[rewriteTo]
	if ok {
		return val
	}
	return ""
}

func isSlash(r rune) bool {
	return r == '/'
}

func ParseAnnotations(origin string, ing *extensions.Ingress) (*Rewrite, error) {
	if ing.GetAnnotations() == nil {
		return &Rewrite{}, errors.New("no annotations present")
	}
	annotations := ingAnnotations(ing.GetAnnotations())
	target := annotations.rewriteTo()
	return &Rewrite{
		Origin: origin,
		Target: target,
	}, nil
}
