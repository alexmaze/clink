// Package sshutil provides a thin SSH/SFTP client used by clink's ssh distribution mode.
package sshutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/alexmaze/clink/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client wraps an SSH connection and an SFTP sub-system.
type Client struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// NewClient dials the server described by server and returns a ready Client.
// Authentication priority:
//  1. Key file (server.Key != "")
//  2. Password (server.Password != "")
//
// The caller is responsible for setting server.Password before calling this
// function if neither Key nor Password is pre-configured (ReadConfig handles
// the interactive prompt).
func NewClient(server *config.SSHServer) (*Client, error) {
	authMethods, err := buildAuthMethods(server)
	if err != nil {
		return nil, fmt.Errorf("sshutil: build auth: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		User:            server.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // acceptable for dotfile management tool
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	sshConn, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("sshutil: dial %s: %w", addr, err)
	}

	sftpConn, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("sshutil: open sftp sub-system: %w", err)
	}

	return &Client{sshClient: sshConn, sftpClient: sftpConn}, nil
}

// Close releases the SFTP sub-system and the underlying SSH connection.
func (c *Client) Close() {
	if c.sftpClient != nil {
		c.sftpClient.Close()
	}
	if c.sshClient != nil {
		c.sshClient.Close()
	}
}

// Exists reports whether remotePath exists on the remote host.
func (c *Client) Exists(remotePath string) (bool, error) {
	_, err := c.sftpClient.Stat(remotePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("sshutil: stat %s: %w", remotePath, err)
}

// MkdirAll creates remotePath and all necessary parents on the remote host.
func (c *Client) MkdirAll(remotePath string) error {
	if err := c.sftpClient.MkdirAll(remotePath); err != nil {
		return fmt.Errorf("sshutil: mkdirall %s: %w", remotePath, err)
	}
	return nil
}

// Upload copies a local file or directory tree to remotePath on the remote host.
// If localPath is a directory every file inside is uploaded recursively.
func (c *Client) Upload(localPath, remotePath string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("sshutil: upload stat %s: %w", localPath, err)
	}

	if info.IsDir() {
		return c.uploadDir(localPath, remotePath)
	}
	return c.uploadFile(localPath, remotePath)
}

// Download copies a remote file or directory tree to localPath.
// If remotePath is a directory every file inside is downloaded recursively.
func (c *Client) Download(remotePath, localPath string) error {
	info, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("sshutil: download stat %s: %w", remotePath, err)
	}

	if info.IsDir() {
		return c.downloadDir(remotePath, localPath)
	}
	return c.downloadFile(remotePath, localPath)
}

// ── private helpers ──────────────────────────────────────────────────────────

func buildAuthMethods(server *config.SSHServer) ([]ssh.AuthMethod, error) {
	if server.Key != "" {
		keyBytes, err := os.ReadFile(server.Key)
		if err != nil {
			return nil, fmt.Errorf("read key file %s: %w", server.Key, err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("parse key file %s: %w", server.Key, err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}

	if server.Password != "" {
		return []ssh.AuthMethod{ssh.Password(server.Password)}, nil
	}

	return nil, fmt.Errorf("no authentication method available for server %s (key and password both empty)", server.Host)
}

func (c *Client) uploadFile(localPath, remotePath string) error {
	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("sshutil: open local %s: %w", localPath, err)
	}
	defer src.Close()

	if err := c.MkdirAll(filepath.Dir(remotePath)); err != nil {
		return err
	}

	dst, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("sshutil: create remote %s: %w", remotePath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("sshutil: copy to remote %s: %w", remotePath, err)
	}
	return nil
}

func (c *Client) uploadDir(localDir, remoteDir string) error {
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("sshutil: readdir %s: %w", localDir, err)
	}

	if err := c.MkdirAll(remoteDir); err != nil {
		return err
	}

	for _, entry := range entries {
		localChild := filepath.Join(localDir, entry.Name())
		remoteChild := remoteDir + "/" + entry.Name()
		if entry.IsDir() {
			if err := c.uploadDir(localChild, remoteChild); err != nil {
				return err
			}
		} else {
			if err := c.uploadFile(localChild, remoteChild); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) downloadFile(remotePath, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("sshutil: mkdirall local %s: %w", filepath.Dir(localPath), err)
	}

	src, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("sshutil: open remote %s: %w", remotePath, err)
	}
	defer src.Close()

	dst, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("sshutil: create local %s: %w", localPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("sshutil: copy from remote %s: %w", remotePath, err)
	}
	return nil
}

func (c *Client) downloadDir(remoteDir, localDir string) error {
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("sshutil: mkdirall local %s: %w", localDir, err)
	}

	entries, err := c.sftpClient.ReadDir(remoteDir)
	if err != nil {
		return fmt.Errorf("sshutil: readdir remote %s: %w", remoteDir, err)
	}

	for _, entry := range entries {
		remoteChild := remoteDir + "/" + entry.Name()
		localChild := filepath.Join(localDir, entry.Name())
		if entry.IsDir() {
			if err := c.downloadDir(remoteChild, localChild); err != nil {
				return err
			}
		} else {
			if err := c.downloadFile(remoteChild, localChild); err != nil {
				return err
			}
		}
	}
	return nil
}
