package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func startPlugin(plugin, pluginOpts, ssAddr string, isServer bool) (newAddr string, err error) {
	logf("starting plugin (%s) with option (%s)....", plugin, pluginOpts)
	freePort, err := getFreePort()
	if err != nil {
		return "", fmt.Errorf("failed to fetch an unused port for plugin (%v)", err)
	}
	localHost := "127.0.0.1"
	ssHost, ssPort, err := net.SplitHostPort(ssAddr)
	if err != nil {
		return "", err
	}
	newAddr = localHost + ":" + freePort
	if isServer {
		if ssHost == "" {
			ssHost = "0.0.0.0"
		}
		logf("plugin (%s) will listen on %s:%s", plugin, ssHost, ssPort)
	} else {
		logf("plugin (%s) will listen on %s:%s", plugin, localHost, freePort)
	}
	err = execPlugin(plugin, pluginOpts, ssHost, ssPort, localHost, freePort)
	return
}

func execPlugin(plugin, pluginOpts, remoteHost, remotePort, localHost, localPort string) error {
	if path, err := findPath(plugin); err != nil {
		return err
	} else {
		plugin = path
	}
	logH := newLogHelper("[" + plugin + "]: ")
	env := append(os.Environ(),
		"SS_REMOTE_HOST="+remoteHost,
		"SS_REMOTE_PORT="+remotePort,
		"SS_LOCAL_HOST="+localHost,
		"SS_LOCAL_PORT="+localPort,
		"SS_PLUGIN_OPTIONS="+pluginOpts,
	)
	cmd := &exec.Cmd{
		Path:   plugin,
		Args:   []string{plugin},
		Env:    env,
		Stdout: logH,
		Stderr: logH,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			logf("plugin exited (%v)\n", err)
			os.Exit(2)
		}
		logf("plugin exited\n")
		os.Exit(0)
	}()
	return nil
}

func findPath(file string) (string, error) {
	skip := []string{"/", "./", "../"}

	for _, p := range skip {
		if strings.HasPrefix(file, p) {
			err := findExecutable(file)
			if err != nil {
				return "", err
			}
			return filepath.Abs(file)
		}
	}

	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	currentExePath := filepath.Join(filepath.Dir(executable), file)

	err = findExecutable(currentExePath)
	if err == nil {
		return currentExePath, nil
	}

	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		path := filepath.Join(dir, file)
		if err := findExecutable(path); err == nil {
			return path, nil
		}
	}
	return "", errors.New("plugin not found!")
}

func findExecutable(file string) error {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := fileInfo.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}

func getFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	port := fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
	l.Close()
	return port, nil
}
