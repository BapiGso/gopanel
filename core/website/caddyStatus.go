package website

import "os/exec"

func startCaddy() error {
	cmd := exec.Command("caddy", "run")
	return cmd.Start()
}

func stopCaddy() error {
	cmd := exec.Command("caddy", "stop")
	return cmd.Run()
}

func restartCaddy() error {
	if err := stopCaddy(); err != nil {
		return err
	}
	return startCaddy()
}
