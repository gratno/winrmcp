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

func app(name string, server config.Server) error {
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

	log.Printf("[%s] exec before command... \n", name)
	for _, v := range server.Before {
		err := client.Command(v)
		if err != nil {
			return fmt.Errorf("client.Command failed! command:%s err:%w", v, err)
		}
	}
	log.Printf("[%s] exec copy task...\n", name)
	err = client.RoboCopy(server)
	if err != nil {
		return fmt.Errorf("client.RoboCopy failed! err:%w", err)
	}
	log.Printf("[%s] exec after command...", name)
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
	for name, server := range config.Conf.Servers {
		wg.Add(1)
		go func(name string) {
			log.Printf("[+]%s start task...\n", name)
			defer wg.Done()
			if err := app(name, server); err != nil {
				log.Printf("[-]%s run app failed! err: %s\n", name, err)
			}
			log.Printf("[+]%s finish task...\n", name)
		}(name)
	}
	wg.Wait()
	log.Println("[+] game over!")
}
