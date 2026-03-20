package services

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"
	"firemail/internal/providers"
	"firemail/internal/sse"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeIMAPClient struct {
	selectedFolders []string
	markReadCalls   [][]uint32
	markUnreadCalls [][]uint32
	moveCalls       []fakeMoveCall
	markReadErr     error
	markUnreadErr   error
	moveErr         error
}

type fakeMoveCall struct {
	UIDs         []uint32
	TargetFolder string
}

func (c *fakeIMAPClient) Connect(context.Context, providers.IMAPClientConfig) error { return nil }
func (c *fakeIMAPClient) Disconnect() error                                         { return nil }
func (c *fakeIMAPClient) IsConnected() bool                                         { return true }
func (c *fakeIMAPClient) ListFolders(context.Context) ([]*providers.FolderInfo, error) {
	return nil, nil
}
func (c *fakeIMAPClient) SelectFolder(_ context.Context, folderName string) (*providers.FolderStatus, error) {
	c.selectedFolders = append(c.selectedFolders, folderName)
	return &providers.FolderStatus{Name: folderName}, nil
}
func (c *fakeIMAPClient) CreateFolder(context.Context, string) error { return nil }
func (c *fakeIMAPClient) DeleteFolder(context.Context, string) error { return nil }
func (c *fakeIMAPClient) RenameFolder(context.Context, string, string) error {
	return nil
}
func (c *fakeIMAPClient) FetchEmails(context.Context, *providers.FetchCriteria) ([]*providers.EmailMessage, error) {
	return nil, nil
}
func (c *fakeIMAPClient) FetchEmailByUID(context.Context, uint32) (*providers.EmailMessage, error) {
	return nil, nil
}
func (c *fakeIMAPClient) FetchEmailHeaders(context.Context, []uint32) ([]*providers.EmailHeader, error) {
	return nil, nil
}
func (c *fakeIMAPClient) MarkAsRead(_ context.Context, uids []uint32) error {
	c.markReadCalls = append(c.markReadCalls, append([]uint32(nil), uids...))
	return c.markReadErr
}
func (c *fakeIMAPClient) MarkAsUnread(_ context.Context, uids []uint32) error {
	c.markUnreadCalls = append(c.markUnreadCalls, append([]uint32(nil), uids...))
	return c.markUnreadErr
}
func (c *fakeIMAPClient) DeleteEmails(context.Context, []uint32) error { return nil }
func (c *fakeIMAPClient) MoveEmails(_ context.Context, uids []uint32, targetFolder string) error {
	c.moveCalls = append(c.moveCalls, fakeMoveCall{
		UIDs:         append([]uint32(nil), uids...),
		TargetFolder: targetFolder,
	})
	return c.moveErr
}
func (c *fakeIMAPClient) CopyEmails(context.Context, []uint32, string) error { return nil }
func (c *fakeIMAPClient) SearchEmails(context.Context, *providers.SearchCriteria) ([]uint32, error) {
	return nil, nil
}
func (c *fakeIMAPClient) GetFolderStatus(context.Context, string) (*providers.FolderStatus, error) {
	return nil, nil
}
func (c *fakeIMAPClient) GetNewEmails(context.Context, string, uint32) ([]*providers.EmailMessage, error) {
	return nil, nil
}
func (c *fakeIMAPClient) GetEmailsInUIDRange(context.Context, string, uint32, uint32) ([]*providers.EmailMessage, error) {
	return nil, nil
}
func (c *fakeIMAPClient) GetAttachment(context.Context, string, uint32, string) (io.ReadCloser, error) {
	return nil, nil
}

type fakeEmailProvider struct {
	imap          *fakeIMAPClient
	connectCalls  int
	disconnects   int
	connectErr    error
	displayName   string
	supportedAuth []string
}

func (p *fakeEmailProvider) GetName() string        { return "custom" }
func (p *fakeEmailProvider) GetDisplayName() string { return "Fake Custom" }
func (p *fakeEmailProvider) GetSupportedAuthMethods() []string {
	if len(p.supportedAuth) == 0 {
		return []string{"password"}
	}
	return p.supportedAuth
}
func (p *fakeEmailProvider) GetProviderInfo() map[string]interface{} { return map[string]interface{}{} }
func (p *fakeEmailProvider) Connect(context.Context, *models.EmailAccount) error {
	p.connectCalls++
	return p.connectErr
}
func (p *fakeEmailProvider) Disconnect() error {
	p.disconnects++
	return nil
}
func (p *fakeEmailProvider) IsConnected() bool     { return true }
func (p *fakeEmailProvider) IsIMAPConnected() bool { return true }
func (p *fakeEmailProvider) IsSMTPConnected() bool { return false }
func (p *fakeEmailProvider) TestConnection(context.Context, *models.EmailAccount) error {
	return nil
}
func (p *fakeEmailProvider) IMAPClient() providers.IMAPClient { return p.imap }
func (p *fakeEmailProvider) SMTPClient() providers.SMTPClient { return nil }
func (p *fakeEmailProvider) OAuth2Client() providers.OAuth2Client {
	return nil
}
func (p *fakeEmailProvider) SendEmail(context.Context, *models.EmailAccount, *providers.OutgoingMessage) error {
	return nil
}
func (p *fakeEmailProvider) SyncEmails(context.Context, *models.EmailAccount, string, uint32) ([]*providers.EmailMessage, error) {
	return nil, nil
}

type emailStateServiceTestEnv struct {
	db        *gorm.DB
	service   *EmailServiceImpl
	publisher *recordingEventPublisher
	provider  *fakeEmailProvider
	user      *models.User
	account   *models.EmailAccount
	inbox     *models.Folder
	work      *models.Folder
}

func setupEmailStateServiceTestEnv(t *testing.T) *emailStateServiceTestEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.EmailGroup{},
		&models.EmailAccount{},
		&models.Folder{},
		&models.Email{},
		&models.Attachment{},
	))

	user := &models.User{
		Username: fmt.Sprintf("email_state_%s", t.Name()),
		Password: "password123",
		Role:     "admin",
		IsActive: true,
	}
	require.NoError(t, db.Create(user).Error)

	account := &models.EmailAccount{
		UserID:       user.ID,
		Name:         "测试邮箱",
		Email:        "tester@example.com",
		Provider:     "custom",
		AuthMethod:   "password",
		IMAPHost:     "imap.example.com",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPSecurity: "STARTTLS",
		Username:     "tester@example.com",
		Password:     "secret",
		IsActive:     true,
	}
	require.NoError(t, db.Create(account).Error)

	inbox := &models.Folder{
		AccountID:    account.ID,
		Name:         "INBOX",
		DisplayName:  "收件箱",
		Type:         models.FolderTypeInbox,
		Path:         "INBOX",
		Delimiter:    "/",
		IsSelectable: true,
		IsSubscribed: true,
	}
	require.NoError(t, db.Create(inbox).Error)

	work := &models.Folder{
		AccountID:    account.ID,
		Name:         "Projects",
		DisplayName:  "项目",
		Type:         models.FolderTypeCustom,
		Path:         "Projects",
		Delimiter:    "/",
		IsSelectable: true,
		IsSubscribed: true,
	}
	require.NoError(t, db.Create(work).Error)

	fakeProvider := &fakeEmailProvider{imap: &fakeIMAPClient{}}
	factory := providers.NewProviderFactory()
	factory.RegisterProvider("custom", func(*config.EmailProviderConfig) providers.EmailProvider {
		return fakeProvider
	})

	publisher := &recordingEventPublisher{}
	service, ok := NewEmailService(db, factory, publisher).(*EmailServiceImpl)
	require.True(t, ok)

	return &emailStateServiceTestEnv{
		db:        db,
		service:   service,
		publisher: publisher,
		provider:  fakeProvider,
		user:      user,
		account:   account,
		inbox:     inbox,
		work:      work,
	}
}

func (env *emailStateServiceTestEnv) createEmail(t *testing.T, folder *models.Folder, uid uint32, subject string, isRead, isDeleted bool) *models.Email {
	t.Helper()

	email := &models.Email{
		AccountID:     env.account.ID,
		FolderID:      &folder.ID,
		MessageID:     fmt.Sprintf("<%s-%d@example.com>", subject, uid),
		UID:           uid,
		Subject:       subject,
		From:          "sender@example.com",
		Date:          time.Now().UTC(),
		TextBody:      "hello",
		HTMLBody:      "<p>hello</p>",
		IsRead:        isRead,
		IsDeleted:     isDeleted,
		HasAttachment: false,
	}
	require.NoError(t, env.db.Create(email).Error)
	return email
}

func findEventByType(events []*sse.Event, eventType sse.EventType) *sse.Event {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i] != nil && events[i].Type == eventType {
			return events[i]
		}
	}
	return nil
}

func flattenUIDCalls(calls [][]uint32) []uint32 {
	var result []uint32
	for _, call := range calls {
		result = append(result, call...)
	}
	return result
}

func TestMarkEmailAsReadSyncsServerAndUpdatesState(t *testing.T) {
	env := setupEmailStateServiceTestEnv(t)
	ctx := context.Background()

	email := env.createEmail(t, env.inbox, 1001, "single-read", false, false)
	require.NoError(t, env.db.Model(env.account).Update("unread_emails", 1).Error)
	require.NoError(t, env.db.Model(env.inbox).Update("unread_emails", 1).Error)

	require.NoError(t, env.service.MarkEmailAsRead(ctx, env.user.ID, email.ID))

	var stored models.Email
	require.NoError(t, env.db.First(&stored, email.ID).Error)
	require.True(t, stored.IsRead)

	var account models.EmailAccount
	require.NoError(t, env.db.First(&account, env.account.ID).Error)
	require.Equal(t, 0, account.UnreadEmails)

	var folder models.Folder
	require.NoError(t, env.db.First(&folder, env.inbox.ID).Error)
	require.Equal(t, 0, folder.UnreadEmails)

	require.Equal(t, []string{"INBOX"}, env.provider.imap.selectedFolders)
	require.Len(t, env.provider.imap.markReadCalls, 1)
	require.Equal(t, []uint32{1001}, env.provider.imap.markReadCalls[0])

	event := findEventByType(env.publisher.events, sse.EventEmailRead)
	require.NotNil(t, event)
	data, ok := event.Data.(*sse.EmailStatusEventData)
	require.True(t, ok)
	require.Equal(t, email.ID, data.EmailID)
	require.NotNil(t, data.IsRead)
	require.True(t, *data.IsRead)
	require.Equal(t, email.FolderID, data.FolderID)
	require.NotNil(t, data.UnreadDelta)
	require.Equal(t, -1, *data.UnreadDelta)
}

func TestMarkEmailAsReadFailsWhenServerWriteFails(t *testing.T) {
	env := setupEmailStateServiceTestEnv(t)
	ctx := context.Background()

	email := env.createEmail(t, env.inbox, 2001, "server-fail", false, false)
	require.NoError(t, env.db.Model(env.account).Update("unread_emails", 1).Error)
	require.NoError(t, env.db.Model(env.inbox).Update("unread_emails", 1).Error)
	env.provider.imap.markReadErr = fmt.Errorf("imap write failed")

	err := env.service.MarkEmailAsRead(ctx, env.user.ID, email.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to mark email as read on server")

	var stored models.Email
	require.NoError(t, env.db.First(&stored, email.ID).Error)
	require.False(t, stored.IsRead)

	var account models.EmailAccount
	require.NoError(t, env.db.First(&account, env.account.ID).Error)
	require.Equal(t, 1, account.UnreadEmails)

	var folder models.Folder
	require.NoError(t, env.db.First(&folder, env.inbox.ID).Error)
	require.Equal(t, 1, folder.UnreadEmails)

	require.Nil(t, findEventByType(env.publisher.events, sse.EventEmailRead))
}

func TestMarkFolderAsReadSyncsRemoteAndPublishesBulkEvent(t *testing.T) {
	env := setupEmailStateServiceTestEnv(t)
	ctx := context.Background()

	readA := env.createEmail(t, env.inbox, 3001, "folder-a", false, false)
	readB := env.createEmail(t, env.inbox, 3002, "folder-b", false, false)
	deleted := env.createEmail(t, env.inbox, 3003, "folder-deleted", false, true)
	require.NoError(t, env.db.Model(env.account).Update("unread_emails", 2).Error)
	require.NoError(t, env.db.Model(env.inbox).Update("unread_emails", 2).Error)

	require.NoError(t, env.service.MarkFolderAsRead(ctx, env.user.ID, env.inbox.ID))

	var refreshedA models.Email
	require.NoError(t, env.db.First(&refreshedA, readA.ID).Error)
	require.True(t, refreshedA.IsRead)

	var refreshedB models.Email
	require.NoError(t, env.db.First(&refreshedB, readB.ID).Error)
	require.True(t, refreshedB.IsRead)

	var refreshedDeleted models.Email
	require.NoError(t, env.db.First(&refreshedDeleted, deleted.ID).Error)
	require.False(t, refreshedDeleted.IsRead)
	require.True(t, refreshedDeleted.IsDeleted)

	require.Len(t, env.provider.imap.markReadCalls, 1)
	require.ElementsMatch(t, []uint32{3001, 3002}, env.provider.imap.markReadCalls[0])

	event := findEventByType(env.publisher.events, sse.EventFolderReadStateChanged)
	require.NotNil(t, event)
	data, ok := event.Data.(*sse.FolderReadStateEventData)
	require.True(t, ok)
	require.Equal(t, env.account.ID, data.AccountID)
	require.Equal(t, env.inbox.ID, data.FolderID)
	require.Equal(t, 2, data.AffectedCount)
}

func TestMarkAccountAsReadSyncsAllFoldersAndPublishesBulkEvent(t *testing.T) {
	env := setupEmailStateServiceTestEnv(t)
	ctx := context.Background()

	env.createEmail(t, env.inbox, 4001, "account-inbox-a", false, false)
	env.createEmail(t, env.inbox, 4002, "account-inbox-b", false, false)
	env.createEmail(t, env.work, 4003, "account-work", false, false)
	require.NoError(t, env.db.Model(env.account).Update("unread_emails", 3).Error)
	require.NoError(t, env.db.Model(env.inbox).Update("unread_emails", 2).Error)
	require.NoError(t, env.db.Model(env.work).Update("unread_emails", 1).Error)

	require.NoError(t, env.service.MarkAccountAsRead(ctx, env.user.ID, env.account.ID))

	require.ElementsMatch(t, []string{"INBOX", "Projects"}, env.provider.imap.selectedFolders)
	require.ElementsMatch(t, []uint32{4001, 4002, 4003}, flattenUIDCalls(env.provider.imap.markReadCalls))

	var account models.EmailAccount
	require.NoError(t, env.db.First(&account, env.account.ID).Error)
	require.Equal(t, 0, account.UnreadEmails)

	var inbox models.Folder
	require.NoError(t, env.db.First(&inbox, env.inbox.ID).Error)
	require.Equal(t, 0, inbox.UnreadEmails)

	var work models.Folder
	require.NoError(t, env.db.First(&work, env.work.ID).Error)
	require.Equal(t, 0, work.UnreadEmails)

	event := findEventByType(env.publisher.events, sse.EventAccountReadStateChanged)
	require.NotNil(t, event)
	data, ok := event.Data.(*sse.AccountReadStateEventData)
	require.True(t, ok)
	require.Equal(t, env.account.ID, data.AccountID)
	require.Equal(t, 3, data.AffectedCount)
}

func TestDeletedEmailsAreExcludedFromMailboxQueries(t *testing.T) {
	env := setupEmailStateServiceTestEnv(t)
	ctx := context.Background()

	visible := env.createEmail(t, env.inbox, 5001, "project visible", false, false)
	deleted := env.createEmail(t, env.inbox, 5002, "project hidden", false, true)

	accountID := env.account.ID
	getResp, err := env.service.GetEmails(ctx, env.user.ID, &GetEmailsRequest{
		AccountID: &accountID,
		Page:      1,
		PageSize:  20,
		SortBy:    "date",
		SortOrder: "desc",
	})
	require.NoError(t, err)
	require.Len(t, getResp.Emails, 1)
	require.Equal(t, visible.ID, getResp.Emails[0].ID)

	searchResp, err := env.service.SearchEmails(ctx, env.user.ID, &SearchEmailsRequest{
		AccountID: &accountID,
		Query:     "project",
		Page:      1,
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Len(t, searchResp.Emails, 1)
	require.Equal(t, visible.ID, searchResp.Emails[0].ID)

	_, err = env.service.GetEmail(ctx, env.user.ID, deleted.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "email not found")
}

func TestDeleteUnreadEmailPublishesFolderAndUnreadDelta(t *testing.T) {
	env := setupEmailStateServiceTestEnv(t)
	ctx := context.Background()

	email := env.createEmail(t, env.inbox, 6001, "delete-me", false, false)
	require.NoError(t, env.db.Model(env.account).Update("unread_emails", 1).Error)
	require.NoError(t, env.db.Model(env.inbox).Update("unread_emails", 1).Error)

	require.NoError(t, env.service.DeleteEmail(ctx, env.user.ID, email.ID))

	event := findEventByType(env.publisher.events, sse.EventEmailDeleted)
	require.NotNil(t, event)
	data, ok := event.Data.(*sse.EmailStatusEventData)
	require.True(t, ok)
	require.Equal(t, email.FolderID, data.FolderID)
	require.NotNil(t, data.UnreadDelta)
	require.Equal(t, -1, *data.UnreadDelta)
}

func TestMoveEmailPublishesStructuredMoveEvent(t *testing.T) {
	env := setupEmailStateServiceTestEnv(t)
	ctx := context.Background()

	email := env.createEmail(t, env.inbox, 7001, "move-me", false, false)
	require.NoError(t, env.db.Model(env.inbox).Update("unread_emails", 1).Error)

	require.NoError(t, env.service.MoveEmail(ctx, env.user.ID, email.ID, env.work.ID))

	require.Len(t, env.provider.imap.moveCalls, 1)
	require.Equal(t, "Projects", env.provider.imap.moveCalls[0].TargetFolder)
	require.Equal(t, []uint32{7001}, env.provider.imap.moveCalls[0].UIDs)

	event := findEventByType(env.publisher.events, sse.EventEmailMoved)
	require.NotNil(t, event)
	data, ok := event.Data.(*sse.EmailMovedEventData)
	require.True(t, ok)
	require.Equal(t, email.ID, data.EmailID)
	require.Equal(t, env.account.ID, data.AccountID)
	require.NotNil(t, data.SourceFolderID)
	require.Equal(t, env.inbox.ID, *data.SourceFolderID)
	require.Equal(t, env.work.ID, data.TargetFolderID)
	require.False(t, data.IsRead)
}
