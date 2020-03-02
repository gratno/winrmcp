package winrmcp

import (
	"fmt"
	"github.com/gratno/winrmcp/config"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dylanmei/iso8601"
	"github.com/masterzen/winrm"
)

type Winrmcp struct {
	client *winrm.Client
	config *Config
}

type Config struct {
	Auth                  Auth
	Https                 bool
	Insecure              bool
	TLSServerName         string
	CACertBytes           []byte
	ConnectTimeout        time.Duration
	OperationTimeout      time.Duration
	MaxOperationsPerShell int
	TransportDecorator    func() winrm.Transporter
}

type Auth struct {
	User     string
	Password string
}

func New(addr string, config *Config) (*Winrmcp, error) {
	endpoint, err := parseEndpoint(addr, config.Https, config.Insecure, config.TLSServerName, config.CACertBytes, config.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = &Config{}
	}

	params := winrm.NewParameters(
		winrm.DefaultParameters.Timeout,
		winrm.DefaultParameters.Locale,
		winrm.DefaultParameters.EnvelopeSize,
	)

	if config.TransportDecorator != nil {
		params.TransportDecorator = config.TransportDecorator
	}

	if config.OperationTimeout.Seconds() > 0 {
		params.Timeout = iso8601.FormatDuration(config.OperationTimeout)
	}
	client, err := winrm.NewClientWithParameters(
		endpoint, config.Auth.User, config.Auth.Password, params)
	return &Winrmcp{client, config}, err
}

// 解决拷贝大文件太慢
func (fs *Winrmcp) RoboCopy(server config.Server) error {
	cp := robocopy{
		client: fs.client,
		server: server,
	}
	err := cp.ensure()
	if err != nil {
		return fmt.Errorf("cp.ensure failed! err:%w", err)
	}
	err = cp.copy()
	if err != nil {
		return fmt.Errorf("cp.copy failed! err:%w", err)
	}
	return nil
}

func (fs *Winrmcp) Copy(fromPath, toPath string) error {
	f, err := os.Open(fromPath)
	if err != nil {
		return fmt.Errorf("Couldn't read file %s: %v", fromPath, err)
	}

	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("Couldn't stat file %s: %v", fromPath, err)
	}

	if !fi.IsDir() {
		return fs.Write(toPath, f)
	} else {
		fw := fileWalker{
			client:  fs.client,
			config:  fs.config,
			toDir:   toPath,
			fromDir: fromPath,
		}
		return filepath.Walk(fromPath, fw.copyFile)
	}
}

func (fs *Winrmcp) Write(toPath string, src io.Reader) error {
	return doCopy(fs.client, fs.config, src, winPath(toPath))
}

func (fs *Winrmcp) List(remotePath string) ([]FileItem, error) {
	return fetchList(fs.client, winPath(remotePath))
}

func (fs *Winrmcp) Client() *winrm.Client {
	return fs.client
}

func (fs *Winrmcp) Command(command string, arguments ...string) error {
	shell, err := fs.client.CreateShell()
	if err != nil {
		return err
	}
	defer shell.Close()
	cmd, err := shell.Execute(command, arguments...)
	if err != nil {
		return err
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
	return nil
}

type fileWalker struct {
	client  *winrm.Client
	config  *Config
	toDir   string
	fromDir string
}

func (fw *fileWalker) copyFile(fromPath string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if !shouldUploadFile(fi) {
		return nil
	}

	hostPath, _ := filepath.Abs(fromPath)
	fromDir, _ := filepath.Abs(fw.fromDir)
	relPath, _ := filepath.Rel(fromDir, hostPath)
	toPath := filepath.Join(fw.toDir, relPath)

	f, err := os.Open(hostPath)
	if err != nil {
		return fmt.Errorf("Couldn't read file %s: %v", fromPath, err)
	}

	return doCopy(fw.client, fw.config, f, winPath(toPath))
}

func shouldUploadFile(fi os.FileInfo) bool {
	// Ignore dir entries and OS X special hidden file
	return !fi.IsDir() && ".DS_Store" != fi.Name()
}
