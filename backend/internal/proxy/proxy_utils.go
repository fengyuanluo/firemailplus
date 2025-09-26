package proxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/proxy"
)

// ProxyConfig 代理配置结构
type ProxyConfig struct {
	Type     string // "none", "http", "socks5"
	Host     string // 代理服务器地址
	Port     int    // 代理服务器端口
	Username string // 用户名（可选）
	Password string // 密码（可选）
}

// CreateDialer 根据代理配置创建网络拨号器
func CreateDialer(config *ProxyConfig) (proxy.Dialer, error) {
	// 如果没有配置代理或代理类型为none，返回直连拨号器
	if config == nil || config.Type == "none" || config.Type == "" {
		return &net.Dialer{
			Timeout: 30 * time.Second,
		}, nil
	}

	proxyAddr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))

	switch config.Type {
	case "socks5":
		return createSOCKS5Dialer(proxyAddr, config.Username, config.Password)
	case "http":
		return createHTTPProxyDialer(proxyAddr, config.Username, config.Password)
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", config.Type)
	}
}

// createSOCKS5Dialer 创建SOCKS5代理拨号器
func createSOCKS5Dialer(proxyAddr, username, password string) (proxy.Dialer, error) {
	var auth *proxy.Auth
	if username != "" {
		auth = &proxy.Auth{
			User:     username,
			Password: password,
		}
	}

	return proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
}

// createHTTPProxyDialer 创建HTTP代理拨号器
func createHTTPProxyDialer(proxyAddr, username, password string) (proxy.Dialer, error) {
	return &httpProxyDialer{
		proxyAddr: proxyAddr,
		username:  username,
		password:  password,
		timeout:   30 * time.Second,
	}, nil
}

// ValidateProxyConfig 验证代理配置
func ValidateProxyConfig(config *ProxyConfig) error {
	if config == nil {
		return nil
	}

	if config.Type == "none" || config.Type == "" {
		return nil
	}

	if config.Host == "" {
		return fmt.Errorf("proxy host is required")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("proxy port must be between 1 and 65535")
	}

	if config.Type != "http" && config.Type != "socks5" {
		return fmt.Errorf("proxy type must be 'http' or 'socks5'")
	}

	return nil
}

// httpProxyDialer HTTP代理拨号器实现
type httpProxyDialer struct {
	proxyAddr string
	username  string
	password  string
	timeout   time.Duration
}

// Dial 通过HTTP代理建立连接
func (d *httpProxyDialer) Dial(network, addr string) (net.Conn, error) {
	// 连接到代理服务器
	conn, err := net.DialTimeout("tcp", d.proxyAddr, d.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy %s: %w", d.proxyAddr, err)
	}

	// 发送HTTP CONNECT请求
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)

	// 添加认证头
	if d.username != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(d.username + ":" + d.password))
		connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
	}

	connectReq += "\r\n"

	// 发送CONNECT请求
	_, err = conn.Write([]byte(connectReq))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send CONNECT request: %w", err)
	}

	// 读取代理响应
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != 200 {
		conn.Close()
		return nil, fmt.Errorf("proxy returned status %d: %s", resp.StatusCode, resp.Status)
	}

	return conn, nil
}
