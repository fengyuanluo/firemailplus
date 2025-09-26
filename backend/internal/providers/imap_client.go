package providers

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	netproxy "golang.org/x/net/proxy"

	"firemail/internal/encoding"
	"firemail/internal/models"
	"firemail/internal/parser"
	"firemail/internal/proxy"
)

// IMAPClientConfig在interface.go中定义

// StandardIMAPClient 标准IMAP客户端实现
type StandardIMAPClient struct {
	client           *client.Client
	connected        bool
	mutex            sync.RWMutex
	conn             net.Conn // 保存底层连接用于超时管理
	readWriteTimeout time.Duration
}

// NewStandardIMAPClient 创建标准IMAP客户端
func NewStandardIMAPClient() *StandardIMAPClient {
	return &StandardIMAPClient{
		connected:        false,
		readWriteTimeout: 60 * time.Second,
	}
}

// Connect 连接到IMAP服务器
func (c *StandardIMAPClient) Connect(ctx context.Context, config IMAPClientConfig) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		return nil
	}

	// 构建服务器地址
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))

	// 设置连接超时
	connectTimeout := 30 * time.Second
	readWriteTimeout := 60 * time.Second

	// 创建代理Dialer
	dialer, err := c.createDialer(config.ProxyConfig, connectTimeout)
	if err != nil {
		return fmt.Errorf("failed to create dialer: %w", err)
	}

	// 添加代理调试信息
	if config.ProxyConfig != nil {
		hasAuth := config.ProxyConfig.Username != ""
		log.Printf("[DEBUG] IMAP connecting via %s proxy: %s:%d (with auth: %v)",
			config.ProxyConfig.Type, config.ProxyConfig.Host, config.ProxyConfig.Port, hasAuth)
	} else {
		log.Printf("[DEBUG] IMAP direct connection (no proxy configured)")
	}

	var imapClient *client.Client

	// 根据安全类型连接
	switch strings.ToUpper(config.Security) {
	case "SSL", "TLS":
		// 直接使用TLS连接
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}

		// 使用代理Dialer进行TLS连接
		conn, err := c.dialTLS(dialer, addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to IMAP server with TLS: %w", err)
		}

		// 设置读写超时
		conn.SetDeadline(time.Now().Add(readWriteTimeout))

		// 创建IMAP客户端
		imapClient, err = client.New(conn)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to create IMAP client: %w", err)
		}

		// 保存连接引用
		c.conn = conn

	case "STARTTLS":
		// 先明文连接，然后升级到TLS
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to IMAP server: %w", err)
		}

		// 设置读写超时
		conn.SetDeadline(time.Now().Add(readWriteTimeout))

		// 创建IMAP客户端
		imapClient, err = client.New(conn)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to create IMAP client: %w", err)
		}

		// 升级到TLS
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}
		err = imapClient.StartTLS(tlsConfig)
		if err != nil {
			imapClient.Close()
			return fmt.Errorf("failed to start TLS: %w", err)
		}

		// 保存连接引用
		c.conn = conn

	case "NONE":
		// 明文连接
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to IMAP server: %w", err)
		}

		// 设置读写超时
		conn.SetDeadline(time.Now().Add(readWriteTimeout))

		// 创建IMAP客户端
		imapClient, err = client.New(conn)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to create IMAP client: %w", err)
		}

		// 保存连接引用
		c.conn = conn

	default:
		return fmt.Errorf("unsupported security type: %s", config.Security)
	}

	// 发送IMAP ID信息（在认证之前）
	// 这对于163等邮箱的可信部分是必需的
	if len(config.IMAPIDInfo) > 0 {
		if err := c.sendIMAPID(imapClient, config.IMAPIDInfo); err != nil {
			log.Printf("Warning: Failed to send IMAP ID: %v", err)
			// 不要因为IMAP ID失败而中断连接，只记录警告
		}
	}

	// 认证
	if config.OAuth2Token != nil {
		// OAuth2认证
		auth := &OAuth2Auth{
			Username: config.Username,
			Token:    config.OAuth2Token.AccessToken,
		}

		err = imapClient.Authenticate(auth)
	} else {
		// 密码认证
		err = imapClient.Login(config.Username, config.Password)
	}

	if err != nil {
		imapClient.Close()
		// 添加详细的错误调试信息
		fmt.Printf("🔐 [IMAP ERROR] Authentication failed: %v\n", err)
		fmt.Printf("🔐 [IMAP ERROR] Error type: %T\n", err)
		fmt.Printf("🔐 [IMAP ERROR] Error string: %s\n", err.Error())
		return fmt.Errorf("IMAP authentication failed: %w", err)
	}

	c.client = imapClient
	c.connected = true

	return nil
}

// Disconnect 断开IMAP连接
func (c *StandardIMAPClient) Disconnect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected || c.client == nil {
		return nil
	}

	err := c.client.Close()
	c.client = nil
	c.conn = nil
	c.connected = false

	return err
}

// RefreshConnectionTimeout 刷新连接超时时间
func (c *StandardIMAPClient) RefreshConnectionTimeout() error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.conn != nil {
		// 重置读写超时
		return c.conn.SetDeadline(time.Now().Add(c.readWriteTimeout))
	}
	return nil
}

// IsConnectionAlive 检查连接是否仍然活跃
func (c *StandardIMAPClient) IsConnectionAlive() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.connected || c.client == nil || c.conn == nil {
		return false
	}

	// 尝试发送NOOP命令来检查连接状态
	err := c.client.Noop()
	return err == nil
}

// sendIMAPID 发送IMAP ID信息
// 这对于163等邮箱的可信部分是必需的
func (c *StandardIMAPClient) sendIMAPID(imapClient *client.Client, idInfo map[string]string) error {
	log.Printf("Sending IMAP ID with info: %v", idInfo)

	// 构建IMAP ID命令的参数列表
	// IMAP ID命令格式：ID (key1 value1 key2 value2 ...)
	var args []interface{}
	for key, value := range idInfo {
		args = append(args, key, value)
	}

	// 创建IMAP命令
	cmd := &imap.Command{
		Name:      "ID",
		Arguments: []interface{}{args}, // 将参数列表作为一个整体传递
	}

	log.Printf("Executing IMAP ID command with %d argument pairs", len(idInfo))

	// 发送命令并等待响应
	status, err := imapClient.Execute(cmd, nil)
	if err != nil {
		log.Printf("Warning: Failed to send IMAP ID command: %v", err)
		// 对于163邮箱，即使ID命令失败也尝试继续连接
		// 因为有些服务器可能不完全支持ID扩展
		return nil
	}

	if status != nil {
		log.Printf("IMAP ID command response: %s", status.Info)
		if status.Type == imap.StatusRespOk {
			log.Printf("✅ IMAP ID command sent successfully")
		} else {
			log.Printf("⚠️  IMAP ID command failed with status: %s", status.Type)
		}
	} else {
		log.Printf("IMAP ID command completed (no status response)")
	}

	return nil
}

// IsConnected 检查是否已连接
func (c *StandardIMAPClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.client != nil
}

// ListFolders 列出文件夹
func (c *StandardIMAPClient) ListFolders(ctx context.Context) ([]*FolderInfo, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("IMAP client not connected")
	}

	// 获取文件夹列表
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.client.List("", "*", mailboxes)
	}()

	var folders []*FolderInfo
	for m := range mailboxes {
		folderType := detectFolderType(m.Name)
		folder := &FolderInfo{
			Name:         m.Name,
			DisplayName:  m.Name,
			Type:         folderType,
			Path:         m.Name,
			Delimiter:    string(m.Delimiter),
			IsSelectable: !contains(m.Attributes, "\\Noselect"),
			IsSubscribed: true, // 默认订阅
		}
		folders = append(folders, folder)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return folders, nil
}

// SelectFolder 选择文件夹
func (c *StandardIMAPClient) SelectFolder(ctx context.Context, folderName string) (*FolderStatus, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("IMAP client not connected")
	}

	mbox, err := c.client.Select(folderName, false)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder %s: %w", folderName, err)
	}

	status := &FolderStatus{
		Name:         folderName,
		TotalEmails:  int(mbox.Messages),
		UnreadEmails: int(mbox.Unseen),
		UIDValidity:  mbox.UidValidity,
		UIDNext:      mbox.UidNext,
	}

	return status, nil
}

// CreateFolder 创建文件夹
func (c *StandardIMAPClient) CreateFolder(ctx context.Context, folderName string) error {
	if !c.IsConnected() {
		return fmt.Errorf("IMAP client not connected")
	}

	return c.client.Create(folderName)
}

// DeleteFolder 删除文件夹
func (c *StandardIMAPClient) DeleteFolder(ctx context.Context, folderName string) error {
	if !c.IsConnected() {
		return fmt.Errorf("IMAP client not connected")
	}

	return c.client.Delete(folderName)
}

// RenameFolder 重命名文件夹
func (c *StandardIMAPClient) RenameFolder(ctx context.Context, oldName, newName string) error {
	if !c.IsConnected() {
		return fmt.Errorf("IMAP client not connected")
	}

	return c.client.Rename(oldName, newName)
}

// GetFolderStatus 获取文件夹状态
func (c *StandardIMAPClient) GetFolderStatus(ctx context.Context, folderName string) (*FolderStatus, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("IMAP client not connected")
	}

	status, err := c.client.Status(folderName, []imap.StatusItem{
		imap.StatusMessages,
		imap.StatusUnseen,
		imap.StatusUidNext,
		imap.StatusUidValidity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get folder status: %w", err)
	}

	return &FolderStatus{
		Name:         folderName,
		TotalEmails:  int(status.Messages),
		UnreadEmails: int(status.Unseen),
		UIDValidity:  status.UidValidity,
		UIDNext:      status.UidNext,
	}, nil
}

// 辅助函数

// detectFolderType 检测文件夹类型
func detectFolderType(name string) string {
	name = strings.ToLower(name)

	if strings.Contains(name, "inbox") || name == "收件箱" {
		return "inbox"
	}
	if strings.Contains(name, "sent") || name == "已发送" || name == "发件箱" {
		return "sent"
	}
	if strings.Contains(name, "draft") || name == "草稿" || name == "草稿箱" {
		return "drafts"
	}
	if strings.Contains(name, "trash") || strings.Contains(name, "deleted") || name == "垃圾箱" || name == "已删除" {
		return "trash"
	}
	if strings.Contains(name, "spam") || strings.Contains(name, "junk") || name == "垃圾邮件" {
		return "spam"
	}

	return "custom"
}

// contains函数已在capabilities.go中定义

// OAuth2Auth OAuth2认证器
// 实现SASL XOAUTH2机制，符合RFC标准和Microsoft/Google的要求
type OAuth2Auth struct {
	Username string
	Token    string
}

// Start 开始OAuth2认证
// 根据实际测试，Outlook IMAP不需要对认证字符串进行base64编码
// 格式: "user=" + userName + "^Aauth=Bearer " + accessToken + "^A^A"
// 其中^A表示Control+A字符(\x01)
func (a *OAuth2Auth) Start() (string, []byte, error) {
	// 构建认证字符串：user=username^Aauth=Bearer token^A^A
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.Username, a.Token)

	return "XOAUTH2", []byte(authString), nil
}

// Next OAuth2认证下一步
// 处理服务器的挑战响应，对于XOAUTH2通常返回空响应
func (a *OAuth2Auth) Next(challenge []byte) ([]byte, error) {
	// 对于SASL XOAUTH2，如果收到挑战（通常是错误信息），
	// 客户端应该发送空响应来完成认证流程
	return nil, nil
}

// FetchEmails 获取邮件列表
func (c *StandardIMAPClient) FetchEmails(ctx context.Context, criteria *FetchCriteria) ([]*EmailMessage, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("IMAP client not connected")
	}

	// 选择文件夹
	_, err := c.client.Select(criteria.FolderName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	// 构建序列集
	var seqSet *imap.SeqSet
	if len(criteria.UIDs) > 0 {
		seqSet = new(imap.SeqSet)
		for _, uid := range criteria.UIDs {
			seqSet.AddNum(uid)
		}
	} else {
		// 获取所有邮件
		seqSet = new(imap.SeqSet)
		seqSet.AddRange(1, 0) // 1:* 表示所有邮件
	}

	// 构建获取项目
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchRFC822Size,
		imap.FetchUid,
	}

	if criteria.IncludeBody {
		items = append(items, imap.FetchRFC822)
	}

	// 获取邮件
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	// 如果包含邮件正文，增加超时时间以防止大邮件被截断
	if criteria.IncludeBody {
		c.RefreshConnectionTimeout()
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("IMAP fetch panic recovered: %v", r)
				done <- fmt.Errorf("IMAP fetch panic: %v", r)
			}
		}()
		done <- c.client.UidFetch(seqSet, items, messages)
	}()

	var emails []*EmailMessage
	for msg := range messages {
		if msg != nil {
			email := convertIMAPMessage(msg, criteria.IncludeBody)
			if email != nil {
				emails = append(emails, email)
			}
		}
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch emails: %w", err)
	}

	return emails, nil
}

// FetchEmailByUID 根据UID获取单个邮件
func (c *StandardIMAPClient) FetchEmailByUID(ctx context.Context, uid uint32) (*EmailMessage, error) {
	criteria := &FetchCriteria{
		UIDs:        []uint32{uid},
		IncludeBody: true,
	}

	emails, err := c.FetchEmails(ctx, criteria)
	if err != nil {
		return nil, err
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("email with UID %d not found", uid)
	}

	return emails[0], nil
}

// FetchEmailHeaders 获取邮件头信息
func (c *StandardIMAPClient) FetchEmailHeaders(ctx context.Context, uids []uint32) ([]*EmailHeader, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("IMAP client not connected")
	}

	seqSet := new(imap.SeqSet)
	for _, uid := range uids {
		seqSet.AddNum(uid)
	}

	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchRFC822Size,
		imap.FetchUid,
	}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.client.UidFetch(seqSet, items, messages)
	}()

	var headers []*EmailHeader
	for msg := range messages {
		header := convertIMAPMessageToHeader(msg)
		headers = append(headers, header)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch email headers: %w", err)
	}

	return headers, nil
}

// MarkAsRead 标记为已读
func (c *StandardIMAPClient) MarkAsRead(ctx context.Context, uids []uint32) error {
	return c.setFlags(uids, []string{"\\Seen"}, true)
}

// MarkAsUnread 标记为未读
func (c *StandardIMAPClient) MarkAsUnread(ctx context.Context, uids []uint32) error {
	return c.setFlags(uids, []string{"\\Seen"}, false)
}

// DeleteEmails 删除邮件
func (c *StandardIMAPClient) DeleteEmails(ctx context.Context, uids []uint32) error {
	// 设置删除标志
	if err := c.setFlags(uids, []string{"\\Deleted"}, true); err != nil {
		return err
	}

	// 立即执行EXPUNGE来永久删除邮件
	return c.client.Expunge(nil)
}

// setFlags 设置邮件标志
func (c *StandardIMAPClient) setFlags(uids []uint32, flags []string, add bool) error {
	if !c.IsConnected() {
		return fmt.Errorf("IMAP client not connected")
	}

	seqSet := new(imap.SeqSet)
	for _, uid := range uids {
		seqSet.AddNum(uid)
	}

	var operation imap.StoreItem
	if add {
		operation = imap.AddFlags
	} else {
		operation = imap.RemoveFlags
	}

	return c.client.UidStore(seqSet, operation, flags, nil)
}

// MoveEmails 移动邮件
func (c *StandardIMAPClient) MoveEmails(ctx context.Context, uids []uint32, targetFolder string) error {
	if !c.IsConnected() {
		return fmt.Errorf("IMAP client not connected")
	}

	seqSet := new(imap.SeqSet)
	for _, uid := range uids {
		seqSet.AddNum(uid)
	}

	return c.client.UidMove(seqSet, targetFolder)
}

// CopyEmails 复制邮件
func (c *StandardIMAPClient) CopyEmails(ctx context.Context, uids []uint32, targetFolder string) error {
	if !c.IsConnected() {
		return fmt.Errorf("IMAP client not connected")
	}

	seqSet := new(imap.SeqSet)
	for _, uid := range uids {
		seqSet.AddNum(uid)
	}

	return c.client.UidCopy(seqSet, targetFolder)
}

// SearchEmails 搜索邮件
func (c *StandardIMAPClient) SearchEmails(ctx context.Context, criteria *SearchCriteria) ([]uint32, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("IMAP client not connected")
	}

	// 选择文件夹
	_, err := c.client.Select(criteria.FolderName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	// 构建搜索条件
	searchCriteria := buildSearchCriteria(criteria)

	// 执行搜索
	uids, err := c.client.UidSearch(searchCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}

	return uids, nil
}

// GetNewEmails 获取新邮件（分批处理）
func (c *StandardIMAPClient) GetNewEmails(ctx context.Context, folderName string, lastUID uint32) ([]*EmailMessage, error) {
	fmt.Printf("📬 [IMAP] GetNewEmails called - folder: %s, lastUID: %d\n", folderName, lastUID)

	if !c.IsConnected() {
		fmt.Printf("❌ [IMAP] Client not connected\n")
		return nil, fmt.Errorf("IMAP client not connected")
	}

	// 选择文件夹
	fmt.Printf("📁 [IMAP] Selecting folder: %s\n", folderName)
	mailbox, err := c.client.Select(folderName, true)
	if err != nil {
		fmt.Printf("❌ [IMAP] Failed to select folder %s: %v\n", folderName, err)
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	fmt.Printf("📊 [IMAP] Folder selected - Messages: %d, Recent: %d, Unseen: %d\n",
		mailbox.Messages, mailbox.Recent, mailbox.Unseen)

	// 首先搜索所有邮件，用于调试
	allSearchCriteria := imap.NewSearchCriteria()
	allSearchCriteria.Uid = new(imap.SeqSet)
	allSearchCriteria.Uid.AddRange(1, 0) // 1:*

	fmt.Printf("🔍 [IMAP] Searching for ALL emails in folder for debugging...\n")
	allUIDs, err := c.client.UidSearch(allSearchCriteria)
	if err != nil {
		fmt.Printf("❌ [IMAP] Failed to search all emails: %v\n", err)
	} else {
		fmt.Printf("📋 [IMAP] ALL UIDs in folder: %v (total: %d)\n", allUIDs, len(allUIDs))
	}

	// 搜索UID大于lastUID的邮件
	searchCriteria := imap.NewSearchCriteria()
	searchCriteria.Uid = new(imap.SeqSet)
	searchCriteria.Uid.AddRange(lastUID+1, 0) // (lastUID+1):*

	fmt.Printf("🔍 [IMAP] Searching for emails with UID > %d\n", lastUID)
	uids, err := c.client.UidSearch(searchCriteria)
	if err != nil {
		fmt.Printf("❌ [IMAP] Failed to search emails: %v\n", err)
		return nil, fmt.Errorf("failed to search new emails: %w", err)
	}

	fmt.Printf("📋 [IMAP] Found %d emails with UID > %d\n", len(uids), lastUID)
	if len(uids) > 0 {
		fmt.Printf("📋 [IMAP] UIDs found: %v\n", uids)
	}

	// 增强的UID恢复机制：检查UID不连续的情况
	if len(uids) == 0 && len(allUIDs) > 0 {
		fmt.Printf("⚠️ [IMAP] No new UIDs found, but ALL UIDs exist. Performing enhanced UID recovery...\n")

		// 检查是否有我们遗漏的UID
		var recoveredUIDs []uint32
		for _, uid := range allUIDs {
			if uid > lastUID {
				fmt.Printf("📧 [IMAP] Found missed UID: %d (> lastUID: %d)\n", uid, lastUID)
				recoveredUIDs = append(recoveredUIDs, uid)
			}
		}

		if len(recoveredUIDs) > 0 {
			fmt.Printf("📋 [IMAP] Recovered %d missed UIDs: %v\n", len(recoveredUIDs), recoveredUIDs)
			uids = recoveredUIDs
		}
	}

	// 额外检查：即使找到了新UID，也检查是否有中间缺失的UID
	if len(allUIDs) > 0 && lastUID > 0 {
		var gapUIDs []uint32
		for _, uid := range allUIDs {
			// 检查lastUID和找到的最小新UID之间是否有缺口
			if uid > lastUID && (len(uids) == 0 || uid < uids[0]) {
				gapUIDs = append(gapUIDs, uid)
			}
		}

		if len(gapUIDs) > 0 {
			fmt.Printf("📋 [IMAP] Found %d UIDs in gaps: %v\n", len(gapUIDs), gapUIDs)
			// 将缺口UID添加到结果中
			uids = append(gapUIDs, uids...)
		}
	}

	if len(uids) == 0 {
		fmt.Printf("✅ [IMAP] No new emails found\n")
		return []*EmailMessage{}, nil
	}

	// 分批处理，每批最多50封邮件
	const batchSize = 50
	var allEmails []*EmailMessage

	fmt.Printf("📦 [IMAP] Processing %d emails in batches of %d\n", len(uids), batchSize)

	for i := 0; i < len(uids); i += batchSize {
		end := i + batchSize
		if end > len(uids) {
			end = len(uids)
		}

		batchUIDs := uids[i:end]
		fmt.Printf("📦 [IMAP] Processing batch %d: UIDs %v\n", i/batchSize+1, batchUIDs)

		// 获取这一批邮件（包含正文内容）
		criteria := &FetchCriteria{
			FolderName:  folderName,
			UIDs:        batchUIDs,
			IncludeBody: true, // 获取完整邮件内容
		}

		batchEmails, err := c.FetchEmails(ctx, criteria)
		if err != nil {
			fmt.Printf("❌ [IMAP] Failed to fetch batch %d: %v\n", i/batchSize+1, err)
			return nil, fmt.Errorf("failed to fetch email batch %d-%d: %w", i, end-1, err)
		}

		fmt.Printf("✅ [IMAP] Successfully fetched %d emails in batch %d\n", len(batchEmails), i/batchSize+1)
		allEmails = append(allEmails, batchEmails...)
	}

	fmt.Printf("✅ [IMAP] GetNewEmails completed - returning %d emails\n", len(allEmails))
	return allEmails, nil
}

// GetEmailsInUIDRange 获取指定UID范围内的邮件
func (c *StandardIMAPClient) GetEmailsInUIDRange(ctx context.Context, folderName string, startUID, endUID uint32) ([]*EmailMessage, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("IMAP client not connected")
	}

	// 选择文件夹
	_, err := c.client.Select(folderName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	// 构建UID范围搜索条件
	searchCriteria := imap.NewSearchCriteria()
	searchCriteria.Uid = new(imap.SeqSet)

	if endUID == 0 {
		// 如果endUID为0，表示到最新邮件
		searchCriteria.Uid.AddRange(startUID, 0) // startUID:*
	} else {
		// 指定范围
		searchCriteria.Uid.AddRange(startUID, endUID) // startUID:endUID
	}

	uids, err := c.client.UidSearch(searchCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails in UID range: %w", err)
	}

	if len(uids) == 0 {
		return []*EmailMessage{}, nil
	}

	// 分批处理，每批最多50封邮件
	const batchSize = 50
	var allEmails []*EmailMessage

	for i := 0; i < len(uids); i += batchSize {
		end := i + batchSize
		if end > len(uids) {
			end = len(uids)
		}

		batchUIDs := uids[i:end]

		// 获取这一批邮件（包含正文内容）
		criteria := &FetchCriteria{
			FolderName:  folderName,
			UIDs:        batchUIDs,
			IncludeBody: true, // 获取完整邮件内容
		}

		batchEmails, err := c.FetchEmails(ctx, criteria)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch email batch %d-%d: %w", i, end-1, err)
		}

		allEmails = append(allEmails, batchEmails...)
	}

	return allEmails, nil
}

// 辅助函数

// convertIMAPMessage 转换IMAP消息为EmailMessage
func convertIMAPMessage(msg *imap.Message, includeBody bool) *EmailMessage {
	email := &EmailMessage{
		UID:     msg.Uid,
		Size:    int64(msg.Size),
		Flags:   msg.Flags,
		Headers: make(map[string][]string),
	}

	// 创建编码助手用于解码邮件头
	encodingHelper := encoding.NewEmailEncodingHelper()

	if msg.Envelope != nil {
		email.MessageID = msg.Envelope.MessageId

		// 解码邮件主题
		email.Subject = encodingHelper.DecodeEmailSubject(msg.Envelope.Subject)
		email.Date = msg.Envelope.Date

		if len(msg.Envelope.From) > 0 {
			email.From = convertIMAPAddressWithDecoding(msg.Envelope.From[0], encodingHelper)
		}

		email.To = convertIMAPAddressesWithDecoding(msg.Envelope.To, encodingHelper)
		email.CC = convertIMAPAddressesWithDecoding(msg.Envelope.Cc, encodingHelper)
		email.BCC = convertIMAPAddressesWithDecoding(msg.Envelope.Bcc, encodingHelper)

		if len(msg.Envelope.ReplyTo) > 0 {
			email.ReplyTo = convertIMAPAddressWithDecoding(msg.Envelope.ReplyTo[0], encodingHelper)
		}
	}

	// 解析邮件正文和附件
	if includeBody {
		// 尝试获取RFC822格式的邮件内容
		if body := msg.GetBody(&imap.BodySectionName{}); body != nil {
			// 使用新的统一解析器
			textBody, htmlBody, attachments := parseEmailBodyUnified(body)
			email.TextBody = textBody
			email.HTMLBody = htmlBody
			email.Attachments = attachments

			// 记录解析结果
			if textBody == "" && htmlBody == "" {
				log.Printf("Warning: No email body content parsed for UID %d", msg.Uid)
			} else {
				log.Printf("Successfully parsed email body for UID %d (text: %d chars, html: %d chars)",
					msg.Uid, len(textBody), len(htmlBody))
			}
		} else {
			log.Printf("Warning: No email body found for UID %d", msg.Uid)
		}
	}

	return email
}

// convertIMAPMessageToHeader 转换IMAP消息为EmailHeader
func convertIMAPMessageToHeader(msg *imap.Message) *EmailHeader {
	header := &EmailHeader{
		UID:   msg.Uid,
		Size:  int64(msg.Size),
		Flags: msg.Flags,
	}

	if msg.Envelope != nil {
		header.MessageID = msg.Envelope.MessageId
		header.Subject = msg.Envelope.Subject
		header.Date = msg.Envelope.Date

		if len(msg.Envelope.From) > 0 {
			header.From = convertIMAPAddress(msg.Envelope.From[0])
		}
	}

	return header
}

// convertIMAPAddress 转换IMAP地址
func convertIMAPAddress(addr *imap.Address) *models.EmailAddress {
	if addr == nil {
		return nil
	}

	return &models.EmailAddress{
		Name:    addr.PersonalName,
		Address: addr.MailboxName + "@" + addr.HostName,
	}
}

// convertIMAPAddressWithDecoding 转换IMAP地址并解码
func convertIMAPAddressWithDecoding(addr *imap.Address, encodingHelper *encoding.EmailEncodingHelper) *models.EmailAddress {
	if addr == nil {
		return nil
	}

	// 解码个人姓名
	decodedName := encodingHelper.DecodeEmailFrom(addr.PersonalName)

	return &models.EmailAddress{
		Name:    decodedName,
		Address: addr.MailboxName + "@" + addr.HostName,
	}
}

// convertIMAPAddressesWithDecoding 转换IMAP地址列表并解码
func convertIMAPAddressesWithDecoding(addrs []*imap.Address, encodingHelper *encoding.EmailEncodingHelper) []*models.EmailAddress {
	var result []*models.EmailAddress
	for _, addr := range addrs {
		if converted := convertIMAPAddressWithDecoding(addr, encodingHelper); converted != nil {
			result = append(result, converted)
		}
	}
	return result
}

// buildSearchCriteria 构建搜索条件
func buildSearchCriteria(criteria *SearchCriteria) *imap.SearchCriteria {
	searchCriteria := imap.NewSearchCriteria()

	if criteria.Subject != "" {
		searchCriteria.Header.Set("Subject", criteria.Subject)
	}

	if criteria.From != "" {
		searchCriteria.Header.Set("From", criteria.From)
	}

	if criteria.To != "" {
		searchCriteria.Header.Set("To", criteria.To)
	}

	if criteria.Body != "" {
		searchCriteria.Text = []string{criteria.Body}
	}

	if criteria.Since != nil {
		searchCriteria.Since = *criteria.Since
	}

	if criteria.Before != nil {
		searchCriteria.Before = *criteria.Before
	}

	if criteria.Seen != nil {
		if *criteria.Seen {
			searchCriteria.WithFlags = []string{"\\Seen"}
		} else {
			searchCriteria.WithoutFlags = []string{"\\Seen"}
		}
	}

	if criteria.Flagged != nil {
		if *criteria.Flagged {
			searchCriteria.WithFlags = append(searchCriteria.WithFlags, "\\Flagged")
		} else {
			searchCriteria.WithoutFlags = append(searchCriteria.WithoutFlags, "\\Flagged")
		}
	}

	return searchCriteria
}

// GetAttachment 获取附件内容
func (c *StandardIMAPClient) GetAttachment(ctx context.Context, folderName string, uid uint32, partID string) (io.ReadCloser, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("IMAP client not connected")
	}

	// 选择文件夹
	_, err := c.client.Select(folderName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	// 构建序列集
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// 构建获取项目 - 获取指定部分的内容
	section := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			Specifier: imap.PartSpecifier(partID),
		},
	}

	items := []imap.FetchItem{imap.FetchItem("BODY[" + partID + "]")}

	// 获取附件内容
	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- c.client.UidFetch(seqSet, items, messages)
	}()

	// 等待消息
	var msg *imap.Message
	select {
	case msg = <-messages:
		if msg == nil {
			return nil, fmt.Errorf("attachment not found")
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// 等待完成
	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch attachment: %w", err)
	}

	// 获取附件内容
	literal := msg.GetBody(section)
	if literal == nil {
		return nil, fmt.Errorf("attachment content not found")
	}

	// 包装literal为ReadCloser
	return io.NopCloser(literal), nil
}

// parseEmailBodyUnified 使用统一解析器解析邮件正文
func parseEmailBodyUnified(body io.Reader) (textBody, htmlBody string, attachments []*AttachmentInfo) {
	if body == nil {
		return "", "", nil
	}

	// 读取邮件内容
	content, err := io.ReadAll(body)
	if err != nil {
		log.Printf("Failed to read email body: %v", err)
		return "", "", nil
	}

	if len(content) == 0 {
		return "", "", nil
	}

	log.Printf("📧 [UNIFIED] Starting unified email parsing, content size: %d bytes", len(content))

	// 使用新的统一解析器
	options := &parser.ParseOptions{
		IncludeAttachmentContent: true,
		MaxAttachmentSize:        25 * 1024 * 1024, // 25MB
		StrictMode:               false,
		MaxErrors:                10,
	}
	unifiedParser := parser.NewUnifiedParser(options)

	parsed, err := unifiedParser.ParseEmail(content)
	if err != nil {
		log.Printf("Warning: Unified parsing failed: %v, falling back to simple parsing", err)
		// 简单回退：尝试将内容作为纯文本处理
		return string(content), "", nil
	}

	// 提取解析结果
	textBody = parsed.TextBody
	htmlBody = parsed.HTMLBody

	// 转换附件格式为兼容格式
	attachments = convertUnifiedAttachmentsToLegacyFormat(parsed.Attachments)

	// 记录解析结果
	log.Printf("Unified parsing completed: text=%d chars, html=%d chars, attachments=%d, errors=%d",
		len(textBody), len(htmlBody), len(attachments), len(parsed.Errors))

	// 记录解析错误（如果有）
	for _, parseErr := range parsed.Errors {
		log.Printf("Parse warning: %v", parseErr)
	}

	return textBody, htmlBody, attachments
}

// convertUnifiedAttachmentsToLegacyFormat 转换统一解析器的附件格式为兼容格式
func convertUnifiedAttachmentsToLegacyFormat(unifiedAttachments []*parser.AttachmentInfo) []*AttachmentInfo {
	var legacyAttachments []*AttachmentInfo

	for _, att := range unifiedAttachments {
		if att == nil {
			continue
		}

		legacyAtt := &AttachmentInfo{
			PartID:      att.PartID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			ContentID:   att.ContentID,
			Disposition: att.Disposition,
			Encoding:    att.Encoding,
			Content:     att.Content,
		}

		legacyAttachments = append(legacyAttachments, legacyAtt)
	}

	return legacyAttachments
}

// createDialer 创建代理Dialer
func (c *StandardIMAPClient) createDialer(proxyConfig *ProxyConfig, timeout time.Duration) (netproxy.Dialer, error) {
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
func (c *StandardIMAPClient) dialTLS(dialer netproxy.Dialer, addr string, tlsConfig *tls.Config) (net.Conn, error) {
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
