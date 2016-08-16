package haproxy

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/golang/glog"
	"github.com/reAsOn2010/ingress/controllers/haproxy/haproxy/config"
)

func (ha *Manager) Start() {
	glog.Info("Starting haproxy process...")
	cmd := exec.Command(haproxyWrapper, "-f", haproxyConfigFilePath, "-p", haproxyPidFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		glog.Errorf("haproxy error: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		glog.Errorf("haproxy error: %v", err)
	}
}

func (ha *Manager) CheckAndReload(cfg config.Configuration, ingressCfg IngressConfig) error {
	ha.reloadRateLimiter.Accept()

	ha.reloadLock.Lock()
	defer ha.reloadLock.Unlock()

	newCfg, err := ha.WriteCfg(cfg, ingressCfg)

	if err != nil {
		return fmt.Errorf("failed to write new haproxy configuration. Avoiding reload: %v", err)
	}

	if newCfg {
		if err := ha.shellOut("kill -s HUP `ps -ef | grep [h]aproxy-systemd-wrapper | awk '{print $2}'`"); err != nil {
			return fmt.Errorf("error reloading haproxy: %v", err)
		}

		glog.Info("change in configuration detected. Reloading...")
	}

	return nil
}

// shellOut executes a command and returns its combined standard output and standard
// error in case of an error in the execution
func (ha *Manager) shellOut(cmd string) error {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		glog.Errorf("failed to execute %v: %v", cmd, string(out))
		return err
	}

	return nil
}
