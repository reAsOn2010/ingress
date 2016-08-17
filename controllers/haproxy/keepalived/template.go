package keepalived

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/golang/glog"
)

var (
	tmplPath = "./keepalived.tmpl"
)

func (k *Manager) loadTemplate() error {
	tmpl, err := template.New("keepalived.tmpl").ParseFiles(tmplPath)
	if err != nil {
		return err
	}
	k.template = tmpl
	return nil
}

func (k *Manager) WriteCfg(cfg *Configuration) (bool, error) {
	if glog.V(3) {
		b, err := json.Marshal(cfg)
		if err != nil {
			glog.Errorf("unexpected error:", err)
		}
		glog.Infof("Keepalived configuration: %v", string(b))
	}
	buffer := new(bytes.Buffer)
	err := k.template.Execute(buffer, cfg)
	if err != nil {
		glog.V(3).Infof("%v", string(buffer.Bytes()))
		return false, err
	}
	changed, err := k.needsReload(buffer)
	if err != nil {
		return false, err
	}
	return changed, nil
}
