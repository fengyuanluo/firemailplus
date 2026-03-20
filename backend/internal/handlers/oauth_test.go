package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"firemail/internal/models"
	"firemail/internal/providers"
	"firemail/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type oauthHandlerTestEnv struct {
	db      *gorm.DB
	handler *Handler
	user    *models.User
}

func setupOAuthHandlerTestEnv(t *testing.T) *oauthHandlerTestEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.EmailGroup{}, &models.EmailAccount{}))

	user := &models.User{
		Username: fmt.Sprintf("oauth_user_%s", t.Name()),
		Password: "password123",
		Role:     "admin",
		IsActive: true,
	}
	require.NoError(t, db.Create(user).Error)

	emailService := services.NewEmailService(db, providers.NewProviderFactory(), nil)

	return &oauthHandlerTestEnv{
		db: db,
		handler: &Handler{
			db:           db,
			emailService: emailService,
		},
		user: user,
	}
}

func (e *oauthHandlerTestEnv) ensureDefaultGroup(t *testing.T) *models.EmailGroup {
	t.Helper()

	group, err := e.handler.emailService.ResolveEmailGroup(context.Background(), e.user.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, group)
	return group
}

func (e *oauthHandlerTestEnv) createGroup(t *testing.T, name string, sortOrder int) *models.EmailGroup {
	t.Helper()

	group := &models.EmailGroup{
		UserID:    e.user.ID,
		Name:      name,
		SortOrder: sortOrder,
		IsDefault: false,
	}
	require.NoError(t, e.db.Create(group).Error)
	return group
}

func (e *oauthHandlerTestEnv) newOAuthAccount(t *testing.T, email string) *models.EmailAccount {
	t.Helper()

	tokenData := &models.OAuth2TokenData{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
		ClientID:     "client-id",
	}

	account := &models.EmailAccount{
		UserID:       e.user.ID,
		Name:         email,
		Email:        email,
		Provider:     "gmail",
		AuthMethod:   "oauth2",
		Username:     email,
		IMAPHost:     "imap.gmail.com",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPSecurity: "STARTTLS",
		IsActive:     true,
		SyncStatus:   "pending",
	}
	require.NoError(t, account.SetOAuth2Token(tokenData))
	return account
}

func TestCreateOAuthAccountWithGroupPersistsResolvedGroup(t *testing.T) {
	env := setupOAuthHandlerTestEnv(t)
	_ = env.ensureDefaultGroup(t)
	workGroup := env.createGroup(t, "工作", 1)
	account := env.newOAuthAccount(t, "persisted@gmail.com")

	persisted, err := env.handler.createOAuthAccountWithGroup(context.Background(), env.user.ID, account, &workGroup.ID)
	require.NoError(t, err)
	require.NotZero(t, persisted.ID)
	require.NotNil(t, persisted.GroupID)
	require.Equal(t, workGroup.ID, *persisted.GroupID)

	var stored models.EmailAccount
	require.NoError(t, env.db.First(&stored, persisted.ID).Error)
	require.NotNil(t, stored.GroupID)
	require.Equal(t, workGroup.ID, *stored.GroupID)
}

func TestCreateOAuthAccountWithGroupRejectsInvalidGroupWithoutCreatingAccount(t *testing.T) {
	env := setupOAuthHandlerTestEnv(t)
	_ = env.ensureDefaultGroup(t)
	account := env.newOAuthAccount(t, "invalid-group@gmail.com")
	invalidGroupID := uint(9999)

	_, err := env.handler.createOAuthAccountWithGroup(context.Background(), env.user.ID, account, &invalidGroupID)
	require.Error(t, err)
	require.True(t, errors.Is(err, errInvalidOAuthAccountGroup))

	var count int64
	require.NoError(t, env.db.Model(&models.EmailAccount{}).Where("user_id = ?", env.user.ID).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestCreateManualOAuth2AccountReturnsConflictOnDuplicate(t *testing.T) {
	env := setupOAuthHandlerTestEnv(t)
	_ = env.ensureDefaultGroup(t)
	existing := env.newOAuthAccount(t, "duplicate@gmail.com")
	require.NoError(t, env.db.Create(existing).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := `{
		"name":"duplicate@gmail.com",
		"email":"duplicate@gmail.com",
		"provider":"gmail",
		"client_id":"client-id",
		"client_secret":"client-secret",
		"refresh_token":"refresh-token"
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/oauth/manual-config", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	c.Request = request
	c.Set("userID", env.user.ID)

	env.handler.CreateManualOAuth2Account(c)

	require.Equal(t, http.StatusConflict, recorder.Code)

	var response ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Contains(t, response.Message, "该邮箱账户已存在")
}
