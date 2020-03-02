package winrmcp

import (
	"fmt"
	"github.com/gratno/winrmcp/config"
	"github.com/masterzen/winrm"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const tempShare = `C:\Share`

type robocopy struct {
	client *winrm.Client
	server config.Server
}

func (rc *robocopy) ensure() error {
	shell, err := rc.client.CreateShell()
	if err != nil {
		return fmt.Errorf("create shell failed! err:%w", err)
	}
	defer shell.Close()
	cmd, err := shell.Execute(winrm.Powershell(fmt.Sprintf(`mkdir %s`, tempShare)))
	if err != nil {
		return fmt.Errorf("execute shell failed! err:%w", err)
	}
	defer cmd.Close()
	var wg sync.WaitGroup
	copyFunc := func(w io.Writer, r io.Reader) {
		defer wg.Done()
		io.Copy(w, r)
	}

	wg.Add(2)
	go copyFunc(os.Stdout, cmd.Stdout)
	go copyFunc(os.Stderr, cmd.Stderr)

	cmd.Wait()
	wg.Wait()
	host := rc.server.Addr
	if i := strings.Index(host, ":"); i > 0 {
		host = host[:i]
	}
	s := fmt.Sprintf(`net use \\%s\%s %s /user:%s`,
		host, strings.Replace(tempShare, ":", "$", 1),
		rc.server.Pass, rc.server.User,
	)
	name, args := splitCommand(s)
	command := exec.Command(name, args...)
	err = command.Run()
	if err != nil {
		return fmt.Errorf("command.Run failed! err:%w", err)
	}
	return nil
}

func (rc *robocopy) copy() error {
	// src file should locate in %share%
	paths := make(map[string]string)
	for _, v := range rc.server.Tasks {
		file := filepath.Base(v.Src)
		temp := filepath.Join(tempShare, file)
		_, err := os.Stat(temp)
		if err != nil {
			return fmt.Errorf("os.Stat failed! path:%s err:%w", temp, err)
		}
		paths[temp] = v.Dst
	}
	host := rc.server.Addr
	if i := strings.Index(host, ":"); i > 0 {
		host = host[:i]
	}

	s := fmt.Sprintf(`robocopy %s \\%s\%s /e /s /xo`,
		tempShare, host, strings.Replace(tempShare, ":", "$", 1),
	)
	name, args := splitCommand(s)
	command := exec.Command(name, args...)
	command.Run()
	time.Sleep(time.Second)
	shell, err := rc.client.CreateShell()
	if err != nil {
		return fmt.Errorf("create shell failed! err:%w", err)
	}
	defer shell.Close()

	var wg sync.WaitGroup
	copyFunc := func(w io.Writer, r io.Reader) {
		defer wg.Done()
		io.Copy(w, r)
	}

	for k, v := range paths {
		cmd, err := shell.Execute(fmt.Sprintf("copy /Y %s %s", k, v))
		if err != nil {
			return fmt.Errorf("execute shell failed! err:%w", err)
		}
		wg.Add(2)
		go copyFunc(os.Stdout, cmd.Stdout)
		go copyFunc(os.Stderr, cmd.Stderr)

		cmd.Wait()
		wg.Wait()
		cmd.Close()
	}
	time.Sleep(time.Second)
	return nil
}

func splitCommand(command string) (name string, args []string) {
	strs := strings.Split(command, " ")
	return strs[0], strs[1:]
}
