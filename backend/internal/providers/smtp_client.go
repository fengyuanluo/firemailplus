package providers

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"firemail/internal/models"
	"firemail/internal/proxy"

	netproxy "golang.org/x/net/proxy"
)

// StandardSMTPClient 标准SMTP客户端实现
type StandardSMTPClient struct {
	client    *smtp.Client
	auth      smtp.Auth
	config    SMTPClientConfig
	connected bool
	mutex     sync.RWMutex
}

// SMTPClientConfig在interface.go中定义

// NewStandardSMTPClient 创建标准SMTP客户端
func NewStandardSMTPClient() *StandardSMTPClient {
	return &StandardSMTPClient{
		connected: false,
	}
}

// Connect 连接到SMTP服务器
func (c *StandardSMTPClient) Connect(ctx context.Context, config SMTPClientConfig) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		return nil
	}

	c.config = config

	// 构建服务器地址
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))

	// 创建代理Dialer
	dialer, err := c.createDialer(config.ProxyConfig, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to create dialer: %w", err)
	}

	// 添加代理调试信息
	if config.ProxyConfig != nil {
		hasAuth := config.ProxyConfig.Username != ""
		log.Printf("[DEBUG] SMTP connecting via %s proxy: %s:%d (with auth: %v)",
			config.ProxyConfig.Type, config.ProxyConfig.Host, config.ProxyConfig.Port, hasAuth)
	} else {
		log.Printf("[DEBUG] SMTP direct connection (no proxy configured)")
	}

	var smtpClient *smtp.Client

	// 根据安全类型连接
	switch strings.ToUpper(config.Security) {
	case "SSL", "TLS":
		// 直接使用TLS连接
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}
		conn, err := c.dialTLS(dialer, addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dial TLS: %w", err)
		}
		smtpClient, err = smtp.NewClient(conn, config.Host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	case "STARTTLS":
		// 先明文连接，然后升级到TLS
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to dial: %w", err)
		}
		smtpClient, err = smtp.NewClient(conn, config.Host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}
		err = smtpClient.StartTLS(tlsConfig)
		if err != nil {
			smtpClient.Close()
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	case "NONE":
		// 明文连接
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to dial: %w", err)
		}
		smtpClient, err = smtp.NewClient(conn, config.Host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	default:
		return fmt.Errorf("unsupported security type: %s", config.Security)
	}

	// 设置认证
	if config.OAuth2Token != nil {
		// OAuth2认证
		c.auth = &OAuth2SMTPAuth{
			Username: config.Username,
			Token:    config.OAuth2Token.AccessToken,
		}
	} else {
		// 密码认证
		c.auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	// 执行认证
	if c.auth != nil {
		// 对于NONE安全类型，需要特殊处理以允许未加密认证
		if strings.ToUpper(config.Security) == "NONE" {
			// 对于未加密连接，我们需要手动处理认证
			// 因为Go的smtp包默认不允许在未加密连接上使用Auth
			if err := c.authenticateUnencrypted(smtpClient, config); err != nil {
				smtpClient.Close()
				return fmt.Errorf("SMTP authentication failed: %w", err)
			}
		} else {
			if err := smtpClient.Auth(c.auth); err != nil {
				smtpClient.Close()
				return fmt.Errorf("SMTP authentication failed: %w", err)
			}
		}
	}

	c.client = smtpClient
	c.connected = true

	return nil
}

// authenticateUnencrypted 在未加密连接上进行认证
func (c *StandardSMTPClient) authenticateUnencrypted(smtpClient *smtp.Client, config SMTPClientConfig) error {
	// 对于未加密连接，很多现代SMTP服务器不允许认证
	// 我们先检查服务器是否支持AUTH扩展
	ext, _ := smtpClient.Extension("AUTH")
	if !ext {
		// 服务器不支持AUTH，不需要认证（如内网SMTP服务器）
		return nil
	}

	// 如果用户名为空，说明不需要认证
	if config.Username == "" {
		return nil
	}

	// 对于需要认证但使用未加密连接的情况，
	// 我们尝试直接使用Go标准库的认证方法
	// 某些内网SMTP服务器（如大学邮箱）允许明文认证

	// 创建认证器
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	// 尝试认证
	// 注意：这会绕过Go标准库的TLS检查，仅用于明确支持明文认证的服务器
	if err := c.forceAuth(smtpClient, auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return nil
}

// forceAuth 强制执行SMTP认证，绕过TLS检查
// 仅用于明确支持明文认证的SMTP服务器
func (c *StandardSMTPClient) forceAuth(smtpClient *smtp.Client, auth smtp.Auth) error {
	// 直接使用自定义认证器，完全绕过Go标准库的TLS检查
	return c.manualPlainAuth(smtpClient, c.config.Username, c.config.Password)
}

// manualPlainAuth 手动执行PLAIN认证
// 这是一个简化的实现，直接跳过TLS检查
func (c *StandardSMTPClient) manualPlainAuth(smtpClient *smtp.Client, username, password string) error {
	// 对于支持明文认证的服务器，我们可以尝试直接登录
	// 这里我们使用一个变通方法：创建一个假的TLS状态

	// 创建一个自定义的认证器，它不检查TLS状态
	customAuth := &PlainAuthNoTLS{
		identity: "",
		username: username,
		password: password,
		host:     c.config.Host,
	}

	return smtpClient.Auth(customAuth)
}

// PlainAuthNoTLS 不检查TLS状态的PLAIN认证器
type PlainAuthNoTLS struct {
	identity, username, password string
	host                         string
}

// Start 开始认证
func (a *PlainAuthNoTLS) Start(server *smtp.ServerInfo) (string, []byte, error) {
	// 不检查TLS状态，直接返回认证信息
	resp := []byte(a.identity + "\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

// Next 认证下一步
func (a *PlainAuthNoTLS) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// 不应该有更多步骤
		return nil, fmt.Errorf("unexpected server challenge")
	}
	return nil, nil
}

// Disconnect 断开SMTP连接
func (c *StandardSMTPClient) Disconnect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected || c.client == nil {
		return nil
	}

	err := c.client.Quit()
	c.client = nil
	c.connected = false

	return err
}

// IsConnected 检查是否已连接
func (c *StandardSMTPClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.client != nil
}

// SendEmail 发送邮件
func (c *StandardSMTPClient) SendEmail(ctx context.Context, message *OutgoingMessage) error {
	if !c.IsConnected() {
		return fmt.Errorf("SMTP client not connected")
	}

	// 构建邮件内容
	emailData, err := c.buildEmailData(message)
	if err != nil {
		return fmt.Errorf("failed to build email data: %w", err)
	}

	// 提取收件人地址
	var recipients []string

	for _, addr := range message.To {
		recipients = append(recipients, addr.Address)
	}
	for _, addr := range message.CC {
		recipients = append(recipients, addr.Address)
	}
	for _, addr := range message.BCC {
		recipients = append(recipients, addr.Address)
	}

	// 发送邮件
	return c.SendRawEmail(ctx, message.From.Address, recipients, emailData)
}

// SendRawEmail 发送原始邮件数据
func (c *StandardSMTPClient) SendRawEmail(ctx context.Context, from string, to []string, data []byte) error {
	if !c.IsConnected() {
		return fmt.Errorf("SMTP client not connected")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 设置发件人
	if err := c.client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// 设置收件人
	for _, recipient := range to {
		if err := c.client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// 发送邮件数据
	writer, err := c.client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer writer.Close()

	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}

	return nil
}

// buildEmailData 构建邮件数据
func (c *StandardSMTPClient) buildEmailData(message *OutgoingMessage) ([]byte, error) {
	var builder strings.Builder

	// 写入邮件头
	c.writeHeaders(&builder, message)

	// 检查是否有附件
	if len(message.Attachments) > 0 {
		// 多部分邮件
		boundary := generateBoundary()
		builder.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		builder.WriteString("\r\n")

		// 邮件正文部分
		c.writeTextPart(&builder, message, boundary)

		// 附件部分
		for _, attachment := range message.Attachments {
			if err := c.writeAttachmentPart(&builder, attachment, boundary); err != nil {
				return nil, fmt.Errorf("failed to write attachment: %w", err)
			}
		}

		// 结束边界
		builder.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	} else {
		// 简单邮件
		c.writeSimpleBody(&builder, message)
	}

	return []byte(builder.String()), nil
}

// writeHeaders 写入邮件头
func (c *StandardSMTPClient) writeHeaders(builder *strings.Builder, message *OutgoingMessage) {
	// 基本头信息
	builder.WriteString(fmt.Sprintf("From: %s\r\n", c.formatAddress(message.From)))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", c.formatAddresses(message.To)))

	if len(message.CC) > 0 {
		builder.WriteString(fmt.Sprintf("Cc: %s\r\n", c.formatAddresses(message.CC)))
	}

	if message.ReplyTo != nil {
		builder.WriteString(fmt.Sprintf("Reply-To: %s\r\n", c.formatAddress(message.ReplyTo)))
	}

	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", mime.QEncoding.Encode("utf-8", message.Subject)))
	builder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	builder.WriteString("MIME-Version: 1.0\r\n")

	// 优先级
	if message.Priority != "" && message.Priority != "normal" {
		switch message.Priority {
		case "high":
			builder.WriteString("X-Priority: 1\r\n")
			builder.WriteString("Importance: high\r\n")
		case "low":
			builder.WriteString("X-Priority: 5\r\n")
			builder.WriteString("Importance: low\r\n")
		}
	}

	// 自定义头
	for key, value := range message.Headers {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
}

// writeSimpleBody 写入简单邮件正文
func (c *StandardSMTPClient) writeSimpleBody(builder *strings.Builder, message *OutgoingMessage) {
	if message.HTMLBody != "" && message.TextBody != "" {
		// 多部分替代内容
		boundary := generateBoundary()
		builder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		builder.WriteString("\r\n")

		// 文本部分
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(message.TextBody))
		builder.WriteString("\r\n")

		// HTML部分
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(message.HTMLBody))
		builder.WriteString("\r\n")

		builder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if message.HTMLBody != "" {
		// 仅HTML
		builder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(message.HTMLBody))
	} else {
		// 仅文本
		builder.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(message.TextBody))
	}
}

// writeTextPart 写入文本部分
func (c *StandardSMTPClient) writeTextPart(builder *strings.Builder, message *OutgoingMessage, boundary string) {
	builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))

	if message.HTMLBody != "" && message.TextBody != "" {
		// 多部分替代内容
		altBoundary := generateBoundary()
		builder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary))
		builder.WriteString("\r\n")

		// 文本部分
		builder.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
		builder.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(message.TextBody))
		builder.WriteString("\r\n")

		// HTML部分
		builder.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
		builder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(message.HTMLBody))
		builder.WriteString("\r\n")

		builder.WriteString(fmt.Sprintf("--%s--\r\n", altBoundary))
	} else if message.HTMLBody != "" {
		// 仅HTML
		builder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(message.HTMLBody))
		builder.WriteString("\r\n")
	} else {
		// 仅文本
		builder.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(message.TextBody))
		builder.WriteString("\r\n")
	}
}

// writeAttachmentPart 写入附件部分
func (c *StandardSMTPClient) writeAttachmentPart(builder *strings.Builder, attachment *OutgoingAttachment, boundary string) error {
	builder.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))

	// 内容类型
	if attachment.ContentType != "" {
		builder.WriteString(fmt.Sprintf("Content-Type: %s\r\n", attachment.ContentType))
	} else {
		builder.WriteString("Content-Type: application/octet-stream\r\n")
	}

	// 内容处置
	disposition := attachment.Disposition
	if disposition == "" {
		disposition = "attachment"
	}

	if attachment.Filename != "" {
		encodedFilename := mime.QEncoding.Encode("utf-8", attachment.Filename)
		builder.WriteString(fmt.Sprintf("Content-Disposition: %s; filename=\"%s\"\r\n", disposition, encodedFilename))
	} else {
		builder.WriteString(fmt.Sprintf("Content-Disposition: %s\r\n", disposition))
	}

	// 内容ID（用于内联附件）
	if attachment.ContentID != "" {
		builder.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", attachment.ContentID))
	}

	builder.WriteString("Content-Transfer-Encoding: base64\r\n")
	builder.WriteString("\r\n")

	// 读取并编码附件内容
	if attachment.Content == nil {
		return fmt.Errorf("attachment %s has nil content", attachment.Filename)
	}

	content, err := io.ReadAll(attachment.Content)
	if err != nil {
		return fmt.Errorf("failed to read attachment content: %w", err)
	}

	// Base64编码
	encoded := encodeBase64(content)
	builder.WriteString(encoded)

	return nil
}

// formatAddress 格式化邮件地址
func (c *StandardSMTPClient) formatAddress(addr *models.EmailAddress) string {
	if addr.Name != "" {
		encodedName := mime.QEncoding.Encode("utf-8", addr.Name)
		return fmt.Sprintf("%s <%s>", encodedName, addr.Address)
	}
	return addr.Address
}

// formatAddresses 格式化邮件地址列表
func (c *StandardSMTPClient) formatAddresses(addrs []*models.EmailAddress) string {
	var formatted []string
	for _, addr := range addrs {
		formatted = append(formatted, c.formatAddress(addr))
	}
	return strings.Join(formatted, ", ")
}

// generateBoundary 生成边界字符串
func generateBoundary() string {
	return fmt.Sprintf("boundary_%d", time.Now().UnixNano())
}

// encodeBase64 Base64编码，每行76个字符
func encodeBase64(data []byte) string {
	const lineLength = 76
	encoded := make([]byte, 0, len(data)*4/3+4)

	// Base64编码表
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	for i := 0; i < len(data); i += 3 {
		var b1, b2, b3 byte
		b1 = data[i]
		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}

		// 编码4个字符
		encoded = append(encoded, base64Table[b1>>2])
		encoded = append(encoded, base64Table[((b1&0x03)<<4)|((b2&0xf0)>>4)])

		if i+1 < len(data) {
			encoded = append(encoded, base64Table[((b2&0x0f)<<2)|((b3&0xc0)>>6)])
		} else {
			encoded = append(encoded, '=')
		}

		if i+2 < len(data) {
			encoded = append(encoded, base64Table[b3&0x3f])
		} else {
			encoded = append(encoded, '=')
		}

		// 每76个字符换行
		if len(encoded)%lineLength == 0 {
			encoded = append(encoded, '\r', '\n')
		}
	}

	// 确保以换行结束
	if len(encoded) > 0 && encoded[len(encoded)-1] != '\n' {
		encoded = append(encoded, '\r', '\n')
	}

	return string(encoded)
}

// encodeQuotedPrintable 对文本进行quoted-printable编码
func encodeQuotedPrintable(text string) string {
	if text == "" {
		return text
	}

	var buf bytes.Buffer
	writer := quotedprintable.NewWriter(&buf)

	_, err := writer.Write([]byte(text))
	if err != nil {
		// 如果编码失败，返回原文本
		return text
	}

	err = writer.Close()
	if err != nil {
		// 如果关闭失败，返回原文本
		return text
	}

	return buf.String()
}

// OAuth2SMTPAuth OAuth2 SMTP认证器
type OAuth2SMTPAuth struct {
	Username string
	Token    string
}

// Start 开始OAuth2认证
func (a *OAuth2SMTPAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "XOAUTH2", []byte(fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.Username, a.Token)), nil
}

// Next OAuth2认证下一步
func (a *OAuth2SMTPAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// 如果服务器返回错误，返回空字节以结束认证
		return []byte{}, nil
	}
	return nil, nil
}

// createDialer 创建代理Dialer
func (c *StandardSMTPClient) createDialer(proxyConfig *ProxyConfig, timeout time.Duration) (netproxy.Dialer, error) {
	// 如果没有代理配置，返回标准Dialer
	if proxyConfig == nil {
		return &net.Dialer{
			Timeout: timeout,
		}, nil
	}

	// 转换为proxy包的ProxyConfig
	proxyConf := proxyConfig.ToProxyConfig()

	// 使用proxy包创建Dialer
	return proxy.CreateDialer(proxyConf)
}

// dialTLS 使用代理进行TLS连接
func (c *StandardSMTPClient) dialTLS(dialer netproxy.Dialer, addr string, tlsConfig *tls.Config) (net.Conn, error) {
	// 先建立到代理的连接
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	// 如果是直连（非代理），直接进行TLS握手
	if _, ok := dialer.(*net.Dialer); ok {
		// 直连情况下，使用tls.Client包装连接
		tlsConn := tls.Client(conn, tlsConfig)
		err = tlsConn.Handshake()
		if err != nil {
			conn.Close()
			return nil, err
		}
		return tlsConn, nil
	}

	// 代理连接情况下，也需要进行TLS握手
	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return tlsConn, nil
}
