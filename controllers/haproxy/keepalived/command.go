package keepalived

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/golang/glog"
)

func (k *Manager) Start() {
	glog.Info("Starting keepalived...")
	cmd := exec.Command("service", "keepalived", "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		glog.Errorf("keepalived error: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		glog.Errorf("keepalived error: %v", err)
	}
}

func (k *Manager) CheckAndReload(cfg *Configuration) error {
	k.reloadRateLimiter.Accept()

	k.reloadLock.Lock()
	defer k.reloadLock.Unlock()

	newCfg, err := k.WriteCfg(cfg)

	if err != nil {
		return fmt.Errorf("failed to write new haproxy configuration. Avoiding reload: %v", err)
	}

	if newCfg {
		if err := k.shellOut("service keepalived reload"); err != nil {
			return fmt.Errorf("error reloading keepalived: %v", err)
		}

		glog.Info("change in configuration detected. Reloading...")
	}

	return nil

}

// shellOut executes a command and returns its combined standard output and standard
// error in case of an error in the execution
func (k *Manager) shellOut(cmd string) error {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		glog.Errorf("failed to execute %v: %v", cmd, string(out))
		return err
	}

	return nil
}
