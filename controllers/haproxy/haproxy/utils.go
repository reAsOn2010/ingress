package haproxy

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/golang/glog"
	"github.com/reAsOn2010/ingress/controllers/haproxy/haproxy/config"
)

func (ha *Manager) needsReload(data *bytes.Buffer) (bool, error) {
	filename := ha.ConfigFile
	in, err := os.Open(filename)
	if err != nil {
		return false, err
	}

	src, err := ioutil.ReadAll(in)
	in.Close()
	if err != nil {
		return false, err
	}

	res := data.Bytes()
	if !bytes.Equal(src, res) {
		err = ioutil.WriteFile(filename, res, 0644)
		if err != nil {
			return false, err
		}

		dData, err := diff(src, res)
		if err != nil {
			glog.Errorf("error computing diff: %s", err)
			return true, nil
		}

		if glog.V(2) {
			glog.Infof("NGINX configuration diff a/%s b/%s\n", filename, filename)
			glog.Infof("%v", string(dData))
		}

		return len(dData) > 0, nil
	}

	return false, nil
}

func diff(b1, b2 []byte) (data []byte, err error) {
	f1, err := ioutil.TempFile("", "")
	if err != nil {
		return
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "")
	if err != nil {
		return
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	data, err = exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		err = nil
	}
	return
}
func (ha *Manager) ReadConfig() config.Configuration {
	return config.NewDefault()
}
