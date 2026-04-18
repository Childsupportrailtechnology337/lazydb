package db

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHTunnelConfig holds the configuration for an SSH tunnel.
type SSHTunnelConfig struct {
	SSHHost     string
	SSHPort     string
	SSHUser     string
	SSHPassword string
	SSHKeyPath  string
	LocalPort   string
	RemoteHost  string
	RemotePort  string
}

// SSHTunnel manages an SSH tunnel lifecycle.
type SSHTunnel struct {
	config   SSHTunnelConfig
	client   *ssh.Client
	listener net.Listener
	quit     chan struct{}
	wg       sync.WaitGroup
}

// NewSSHTunnel creates a new SSHTunnel from the given config.
func NewSSHTunnel(config SSHTunnelConfig) *SSHTunnel {
	if config.SSHPort == "" {
		config.SSHPort = "22"
	}
	if config.LocalPort == "" {
		config.LocalPort = "0"
	}
	if config.RemoteHost == "" {
		config.RemoteHost = "127.0.0.1"
	}
	return &SSHTunnel{
		config: config,
		quit:   make(chan struct{}),
	}
}

// Start establishes the SSH connection and starts forwarding connections
// from the local port to the remote host:port.
func (t *SSHTunnel) Start() error {
	authMethods, err := t.buildAuthMethods()
	if err != nil {
		return fmt.Errorf("ssh auth: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            t.config.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // user-controlled tunnel
	}

	sshAddr := net.JoinHostPort(t.config.SSHHost, t.config.SSHPort)
	client, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return fmt.Errorf("ssh dial %s: %w", sshAddr, err)
	}
	t.client = client

	localAddr := net.JoinHostPort("127.0.0.1", t.config.LocalPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		client.Close()
		return fmt.Errorf("listen on %s: %w", localAddr, err)
	}
	t.listener = listener

	t.wg.Add(1)
	go t.acceptLoop()

	return nil
}

// Stop closes the tunnel, stopping all forwarded connections.
func (t *SSHTunnel) Stop() error {
	close(t.quit)

	var firstErr error
	if t.listener != nil {
		if err := t.listener.Close(); err != nil {
			firstErr = err
		}
	}

	t.wg.Wait()

	if t.client != nil {
		if err := t.client.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// LocalAddr returns the local address the tunnel is listening on,
// e.g. "127.0.0.1:54321".
func (t *SSHTunnel) LocalAddr() string {
	if t.listener == nil {
		return ""
	}
	return t.listener.Addr().String()
}

func (t *SSHTunnel) acceptLoop() {
	defer t.wg.Done()
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.quit:
				return
			default:
				continue
			}
		}
		t.wg.Add(1)
		go t.forward(conn)
	}
}

func (t *SSHTunnel) forward(local net.Conn) {
	defer t.wg.Done()
	defer local.Close()

	remoteAddr := net.JoinHostPort(t.config.RemoteHost, t.config.RemotePort)
	remote, err := t.client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remote.Close()

	done := make(chan struct{}, 2)
	copyConn := func(dst, src net.Conn) {
		io.Copy(dst, src) //nolint:errcheck
		done <- struct{}{}
	}
	go copyConn(remote, local)
	go copyConn(local, remote)

	select {
	case <-done:
	case <-t.quit:
	}
}

func (t *SSHTunnel) buildAuthMethods() ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// Key-based auth
	if t.config.SSHKeyPath != "" {
		keyBytes, err := os.ReadFile(t.config.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", t.config.SSHKeyPath, err)
		}
		var signer ssh.Signer
		if t.config.SSHPassword != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(t.config.SSHPassword))
		} else {
			signer, err = ssh.ParsePrivateKey(keyBytes)
		}
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	// SSH agent
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		conn, err := net.Dial("unix", sock)
		if err == nil {
			agentClient := agent.NewClient(conn)
			methods = append(methods, ssh.PublicKeysCallback(agentClient.Signers))
		}
	}

	// Password auth (only when no key path is set, or as fallback)
	if t.config.SSHPassword != "" && t.config.SSHKeyPath == "" {
		methods = append(methods, ssh.Password(t.config.SSHPassword))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no SSH authentication method configured")
	}
	return methods, nil
}
