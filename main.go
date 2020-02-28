package main

import (
	"fmt"
	"github.com/gratno/winrmcp/config"
	"github.com/masterzen/winrm"
	"log"
	"sync"
	"time"

	"github.com/gratno/winrmcp/winrmcp"
)

func app(thread int, server config.Server) error {
	opTimeout, err := time.ParseDuration(server.OpTimeout)
	if err != nil {
		return fmt.Errorf("time.ParseDuration failed! err:%w", err)
	}

	client, err := winrmcp.New(server.Addr, &winrmcp.Config{
		Auth:                  winrmcp.Auth{User: server.User, Password: server.Pass},
		Https:                 server.Https,
		Insecure:              server.Insecure,
		OperationTimeout:      opTimeout,
		MaxOperationsPerShell: server.MaxOpsPerShell,
		TransportDecorator: func() winrm.Transporter {
			return &winrm.ClientNTLM{}
		},
	})
	if err != nil {
		return fmt.Errorf("winrmcp.New failed! err:%w", err)
	}

	log.Printf("[%d] exec before command... \n", thread)
	for _, v := range server.Before {
		err := client.Command(v)
		if err != nil {
			return fmt.Errorf("client.Command failed! command:%s err:%w", v, err)
		}
	}
	log.Printf("[%d] exec copy task...\n", thread)
	for _, v := range server.Tasks {
		err := client.Copy(v.Src, v.Dst)
		if err != nil {
			return fmt.Errorf("client.Copy failed! err:%w", err)
		}
	}
	log.Printf("[%d] exec after command...", thread)
	for _, v := range server.After {
		err := client.Command(v)
		if err != nil {
			return fmt.Errorf("client.Command failed! command:%s err:%w", v, err)
		}
	}
	return nil
}

func main() {
	log.SetFlags(log.LstdFlags)
	config.Parse()
	wg := sync.WaitGroup{}
	for i, server := range config.Conf.Servers {
		wg.Add(1)
		go func(i int) {
			log.Printf("[+]server-%d start task...\n", i)
			defer wg.Done()
			if err := app(i, server); err != nil {
				log.Printf("[-]server-%d run app failed! err: %s\n", i, err)
			}
			log.Printf("[+]server-%d finish task...\n", i)
		}(i)
	}
	wg.Wait()
	log.Println("[+] game over!")
}
