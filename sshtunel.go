package utils

import (
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHTunel struct {
	Address       string
	RemoteAddress string
	LocalAddress  string
	Config        *ssh.ClientConfig
}

func NewSSHTunel(address, usename, password string, remoteAddress, localAddress string) *SSHTunel {
	// 设置SSH配置
	tunel := &SSHTunel{
		Address:       address,
		RemoteAddress: remoteAddress,
		LocalAddress:  localAddress,
		Config: &ssh.ClientConfig{
			// 服务器用户名
			User: usename,
			Auth: []ssh.AuthMethod{
				// 服务器密码
				ssh.Password(password),
			},
			Timeout: 30 * time.Second,
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		},
	}
	return tunel
}

func (tunel *SSHTunel) Serve() error {
	// 设置本地监听器，格式：地址:端口
	localListener, err := net.Listen("tcp", tunel.LocalAddress)
	if err != nil {
		return err
	}
	for {
		localConn, err := localListener.Accept()
		if err != nil {
			return err
		}
		go tunel.forward(localConn)
	}
}

// 转发
func (tunel *SSHTunel) forward(localConn net.Conn) error {
	defer localConn.Close()
	// 设置服务器地址，格式：地址:端口
	sshclient, err := ssh.Dial("tcp", tunel.Address, tunel.Config)
	if err != nil {
		return err
	}
	defer sshclient.Close()
	// 设置远程地址，格式：地址:端口（请在服务器通过 ifconfig 查看地址）
	sshConn, err := sshclient.Dial("tcp", tunel.RemoteAddress)
	if err != nil {
		return err
	}
	defer sshConn.Close()
	waiter := &sync.WaitGroup{}
	waiter.Add(2)
	// 将localConn.Reader复制到sshConn.Writer
	go func() {
		_, err = io.Copy(sshConn, localConn)
		waiter.Done()
	}()
	// 将sshConn.Reader复制到localConn.Writer
	go func() {
		_, err = io.Copy(localConn, sshConn)
		waiter.Done()
	}()
	waiter.Wait()
	return err
}
