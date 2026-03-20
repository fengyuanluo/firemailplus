package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"firemail/internal/models"
	"firemail/internal/providers"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type emailGroupServiceTestEnv struct {
	db      *gorm.DB
	service *EmailServiceImpl
	user    *models.User
}

func setupEmailGroupServiceTestEnv(t *testing.T) *emailGroupServiceTestEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&models.User{}, &models.EmailGroup{}, &models.EmailAccount{}))

	user := &models.User{
		Username: fmt.Sprintf("user_%s", t.Name()),
		Password: "password123",
		Role:     "admin",
		IsActive: true,
	}
	require.NoError(t, db.Create(user).Error)

	service, ok := NewEmailService(db, providers.NewProviderFactory(), nil).(*EmailServiceImpl)
	require.True(t, ok)

	return &emailGroupServiceTestEnv{
		db:      db,
		service: service,
		user:    user,
	}
}

func (e *emailGroupServiceTestEnv) ensureDefaultGroup(t *testing.T) *models.EmailGroup {
	t.Helper()

	group, err := e.service.ResolveEmailGroup(context.Background(), e.user.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, group)
	return group
}

func (e *emailGroupServiceTestEnv) createGroupRecord(t *testing.T, name string, sortOrder int, isDefault bool) *models.EmailGroup {
	t.Helper()

	group := &models.EmailGroup{
		UserID:    e.user.ID,
		Name:      name,
		SortOrder: sortOrder,
		IsDefault: isDefault,
	}
	require.NoError(t, e.db.Create(group).Error)
	return group
}

func (e *emailGroupServiceTestEnv) createAccountRecord(t *testing.T, email string, groupID *uint) *models.EmailAccount {
	t.Helper()

	providerConfig := providers.NewProviderFactory().GetProviderConfig("qq")
	require.NotNil(t, providerConfig)

	account := &models.EmailAccount{
		UserID:       e.user.ID,
		Name:         email,
		Email:        email,
		Provider:     "qq",
		AuthMethod:   "password",
		GroupID:      groupID,
		Username:     email,
		Password:     "auth-code",
		IMAPHost:     providerConfig.IMAPHost,
		IMAPPort:     providerConfig.IMAPPort,
		IMAPSecurity: providerConfig.IMAPSecurity,
		SMTPHost:     providerConfig.SMTPHost,
		SMTPPort:     providerConfig.SMTPPort,
		SMTPSecurity: providerConfig.SMTPSecurity,
		IsActive:     true,
		SyncStatus:   "pending",
	}
	require.NoError(t, e.db.Create(account).Error)
	return account
}

func reloadAccount(t *testing.T, db *gorm.DB, accountID uint) *models.EmailAccount {
	t.Helper()

	var account models.EmailAccount
	require.NoError(t, db.First(&account, accountID).Error)
	return &account
}

func TestOptionalGroupIDUnmarshal(t *testing.T) {
	t.Run("missing field keeps unset state", func(t *testing.T) {
		var req UpdateEmailAccountRequest
		require.NoError(t, json.Unmarshal([]byte(`{}`), &req))
		require.False(t, req.GroupID.Set)
		require.Nil(t, req.GroupID.Value)
	})

	t.Run("explicit null maps to default-group intent", func(t *testing.T) {
		var req UpdateEmailAccountRequest
		require.NoError(t, json.Unmarshal([]byte(`{"group_id":null}`), &req))
		require.True(t, req.GroupID.Set)
		require.Nil(t, req.GroupID.Value)
	})

	t.Run("explicit number maps to target group", func(t *testing.T) {
		var req UpdateEmailAccountRequest
		require.NoError(t, json.Unmarshal([]byte(`{"group_id":12}`), &req))
		require.True(t, req.GroupID.Set)
		require.NotNil(t, req.GroupID.Value)
		require.Equal(t, uint(12), *req.GroupID.Value)
	})
}

func TestUpdateEmailAccountGroupIDTriState(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	defaultGroup := env.ensureDefaultGroup(t)
	workGroup := env.createGroupRecord(t, "工作", 1, false)
	personalGroup := env.createGroupRecord(t, "私人", 2, false)
	account := env.createAccountRecord(t, "tri-state@qq.com", &workGroup.ID)

	t.Run("omitted group_id keeps existing group", func(t *testing.T) {
		name := "tri-state-renamed"
		updated, err := env.service.UpdateEmailAccount(ctx, env.user.ID, account.ID, &UpdateEmailAccountRequest{
			Name: &name,
		})
		require.NoError(t, err)
		require.NotNil(t, updated.GroupID)
		require.Equal(t, workGroup.ID, *updated.GroupID)
	})

	t.Run("explicit null moves account back to default group", func(t *testing.T) {
		updated, err := env.service.UpdateEmailAccount(ctx, env.user.ID, account.ID, &UpdateEmailAccountRequest{
			GroupID: OptionalGroupID{Set: true, Value: nil},
		})
		require.NoError(t, err)
		require.NotNil(t, updated.GroupID)
		require.Equal(t, defaultGroup.ID, *updated.GroupID)
	})

	t.Run("explicit number moves account to selected group", func(t *testing.T) {
		targetID := personalGroup.ID
		updated, err := env.service.UpdateEmailAccount(ctx, env.user.ID, account.ID, &UpdateEmailAccountRequest{
			GroupID: OptionalGroupID{Set: true, Value: &targetID},
		})
		require.NoError(t, err)
		require.NotNil(t, updated.GroupID)
		require.Equal(t, personalGroup.ID, *updated.GroupID)
	})
}

func TestCreateEmailAccountRejectsDuplicateEmailProvider(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	_ = env.ensureDefaultGroup(t)
	env.createAccountRecord(t, "duplicate@qq.com", nil)

	_, err := env.service.CreateEmailAccount(ctx, env.user.ID, &CreateEmailAccountRequest{
		Name:       "duplicate@qq.com",
		Email:      "duplicate@qq.com",
		Provider:   "qq",
		AuthMethod: "password",
		Password:   "auth-code",
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrEmailAccountAlreadyExists))
}

func TestCreateEmailGroupFirstCustomReturnsFreshDefaultState(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	oldDefault := env.ensureDefaultGroup(t)

	created, err := env.service.CreateEmailGroup(ctx, env.user.ID, &CreateEmailGroupRequest{
		Name: "工作",
	})
	require.NoError(t, err)
	require.True(t, created.IsDefault)

	var freshCreated models.EmailGroup
	require.NoError(t, env.db.First(&freshCreated, created.ID).Error)
	require.True(t, freshCreated.IsDefault)

	var freshOldDefault models.EmailGroup
	require.NoError(t, env.db.First(&freshOldDefault, oldDefault.ID).Error)
	require.False(t, freshOldDefault.IsDefault)
}

func TestSetDefaultEmailGroupMovesOldDefaultAndUngroupedAccounts(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	oldDefault := env.ensureDefaultGroup(t)
	newDefault := env.createGroupRecord(t, "工作", 1, false)
	otherGroup := env.createGroupRecord(t, "其他", 2, false)

	oldDefaultAccount := env.createAccountRecord(t, "old-default@qq.com", &oldDefault.ID)
	nilGroupAccount := env.createAccountRecord(t, "nil-group@qq.com", nil)
	otherGroupAccount := env.createAccountRecord(t, "other-group@qq.com", &otherGroup.ID)

	updatedGroup, err := env.service.SetDefaultEmailGroup(ctx, env.user.ID, newDefault.ID)
	require.NoError(t, err)
	require.True(t, updatedGroup.IsDefault)

	require.Equal(t, newDefault.ID, *reloadAccount(t, env.db, oldDefaultAccount.ID).GroupID)
	require.Equal(t, newDefault.ID, *reloadAccount(t, env.db, nilGroupAccount.ID).GroupID)
	require.Equal(t, otherGroup.ID, *reloadAccount(t, env.db, otherGroupAccount.ID).GroupID)
}

func TestDeleteEmailGroupMovesAccountsBackToDefault(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	defaultGroup := env.ensureDefaultGroup(t)
	workGroup := env.createGroupRecord(t, "工作", 1, false)
	account := env.createAccountRecord(t, "delete-group@qq.com", &workGroup.ID)

	require.NoError(t, env.service.DeleteEmailGroup(ctx, env.user.ID, workGroup.ID))

	reloadedAccount := reloadAccount(t, env.db, account.ID)
	require.NotNil(t, reloadedAccount.GroupID)
	require.Equal(t, defaultGroup.ID, *reloadedAccount.GroupID)

	var remaining models.EmailGroup
	err := env.db.First(&remaining, workGroup.ID).Error
	require.Error(t, err)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
