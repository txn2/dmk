package tunnel

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/cjimti/migration-kit/cfg"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Manager handles the collection of tunnels
type Manager struct {
	// a map of of machine names to drivers
	drivers map[string]*SSHTunnel
}

// Tunnel opens the specified tunnel if it is not
// alreay open.
func (tm *Manager) Tunnel(tunnelCfg cfg.Tunnel) error {
	// TODO: close connection when we are done.
	// see: https://stackoverflow.com/questions/12741386/how-to-know-tcp-connection-is-closed-in-golang-net-package

	// already running?
	if _, ok := tm.drivers[tunnelCfg.Component.MachineName]; ok {
		return nil
	}

	sshConfig := &ssh.ClientConfig{
		User: tunnelCfg.TunnelAuth.User,
		Auth: []ssh.AuthMethod{
			SSHAgent(),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	tunnel := &SSHTunnel{
		Config: sshConfig,
		Local: &Endpoint{
			Host: tunnelCfg.Local.Host,
			Port: tunnelCfg.Local.Port,
		},
		Server: &Endpoint{
			Host: tunnelCfg.Server.Host,
			Port: tunnelCfg.Server.Port,
		},
		Remote: &Endpoint{
			Host: tunnelCfg.Remote.Host,
			Port: tunnelCfg.Remote.Port,
		},
	}

	// @TODO: add error chanel and close channel
	go func() {
		tunnel.Start()
		tm.drivers[tunnelCfg.Component.MachineName] = tunnel
	}()

	return nil
}

// Endpoint contains a host and port tunnel endpoint.
type Endpoint struct {
	Host string
	Port int
}

// String returns a formatted string describing the Endpoint
func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

// SSHTunnel holds pointers to Local, Server and Remote endpoints
// and an ssh configuration.
type SSHTunnel struct {
	Local  *Endpoint
	Server *Endpoint
	Remote *Endpoint

	Config *ssh.ClientConfig
}

// Start an ssh tunnel
func (tunnel *SSHTunnel) Start() error {
	listener, err := net.Listen("tcp", tunnel.Local.String())
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go tunnel.forward(conn)
	}
}

// forward a connection
func (tunnel *SSHTunnel) forward(localConn net.Conn) {

	serverConn, err := ssh.Dial("tcp", tunnel.Server.String(), tunnel.Config)
	if err != nil {
		fmt.Printf("Server dial error: %s\n", err)
		return
	}

	remoteConn, err := serverConn.Dial("tcp", tunnel.Remote.String())
	if err != nil {
		fmt.Printf("Remote dial error: %s\n", err)
		return
	}

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

// SSHAgent uses the exposed ssh agent from SSH_AUTH_SOCK
// TODO: Allow an ssh key to be specified (for automated / server runs)
func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}

	return nil
}
