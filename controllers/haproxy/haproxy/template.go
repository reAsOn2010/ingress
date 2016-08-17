package haproxy

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"text/template"

	"github.com/fatih/structs"
	"github.com/golang/glog"
	"github.com/reAsOn2010/ingress/controllers/haproxy/haproxy/config"
)

var (
	camelRegexp = regexp.MustCompile("[0-9A-Za-z]+")
	// TODO: update here
	tmplPath = "./haproxy.tmpl"
	funcMap  = template.FuncMap{
		"empty": func(input interface{}) bool {
			check, ok := input.(string)
			if ok {
				return len(check) == 0
			}
			return true
		},
		"replaceDot": replaceDot,
		// "buildLocation":       buildLocation,
		// "buildProxyPass":      buildProxyPass,
		// "buildRateLimitZones": buildRateLimitZones,
		// "buildRateLimit":      buildRateLimit,
	}
)

func (ha *Manager) loadTemplate() error {
	tmpl, err := template.New("haproxy.tmpl").Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		return err
	}
	ha.template = tmpl
	return nil
}

func (ha *Manager) WriteCfg(cfg config.Configuration, ingressCfg IngressConfig) (bool, error) {
	conf := make(map[string]interface{})
	conf["services"] = ingressCfg
	conf["cfg"] = fixKeyNames(structs.Map(cfg))
	if glog.V(3) {
		b, err := json.Marshal(conf)
		if err != nil {
			glog.Errorf("unexpected error:", err)
		}
		glog.Infof("Haproxy configuration: %v", string(b))
	}
	buffer := new(bytes.Buffer)
	err := ha.template.Execute(buffer, conf)
	if err != nil {
		glog.V(3).Infof("%v", string(buffer.Bytes()))
		return false, err
	}
	changed, err := ha.needsReload(buffer)
	if err != nil {
		return false, err
	}

	return changed, nil
}

func fixKeyNames(data map[string]interface{}) map[string]interface{} {
	fixed := make(map[string]interface{})
	for k, v := range data {
		fixed[toCamelCase(k)] = v
	}

	return fixed
}

func toCamelCase(src string) string {
	byteSrc := []byte(src)
	chunks := camelRegexp.FindAll(byteSrc, -1)
	for idx, val := range chunks {
		if idx > 0 {
			chunks[idx] = bytes.Title(val)
		}
	}
	return string(bytes.Join(chunks, nil))
}

func replaceDot(str string) string {
	return strings.Replace(str, ".", "-", -1)
}
