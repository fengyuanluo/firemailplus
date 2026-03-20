package services

import (
	"context"
	"errors"
	"testing"

	"firemail/internal/models"

	"github.com/stretchr/testify/require"
)

func (e *emailGroupServiceTestEnv) createSystemGroupRecord(t *testing.T, name string, sortOrder int, isDefault bool, systemKey string) *models.EmailGroup {
	t.Helper()

	group := &models.EmailGroup{
		UserID:    e.user.ID,
		Name:      name,
		SortOrder: sortOrder,
		IsDefault: isDefault,
		SystemKey: &systemKey,
	}
	require.NoError(t, e.db.Create(group).Error)
	return group
}

func TestRepairEmailGroupInvariantsForUserCreatesDefaultPlaceholderAndRepairsUngroupedAccounts(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	account := env.createAccountRecord(t, "orphan@qq.com", nil)

	require.NoError(t, RepairEmailGroupInvariantsForUser(ctx, env.db, env.user.ID))

	var groups []models.EmailGroup
	require.NoError(t, env.db.Where("user_id = ?", env.user.ID).Find(&groups).Error)
	require.Len(t, groups, 1)
	require.True(t, groups[0].IsDefault)
	require.NotNil(t, groups[0].SystemKey)
	require.Equal(t, models.EmailGroupSystemKeyDefaultPlaceholder, *groups[0].SystemKey)

	reloaded := reloadAccount(t, env.db, account.ID)
	require.NotNil(t, reloaded.GroupID)
	require.Equal(t, groups[0].ID, *reloaded.GroupID)
}

func TestRepairEmailGroupInvariantsForUserBackfillsHistoricalPlaceholderSystemKey(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	historicalDefault := env.createGroupRecord(t, "未分", 0, true)

	require.NoError(t, RepairEmailGroupInvariantsForUser(ctx, env.db, env.user.ID))

	var reloaded models.EmailGroup
	require.NoError(t, env.db.First(&reloaded, historicalDefault.ID).Error)
	require.NotNil(t, reloaded.SystemKey)
	require.Equal(t, models.EmailGroupSystemKeyDefaultPlaceholder, *reloaded.SystemKey)
	require.True(t, reloaded.IsDefault)
}

func TestRepairEmailGroupInvariantsForUserBackfillsDemotedHistoricalPlaceholder(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	currentDefault := env.createGroupRecord(t, "工作", 0, true)
	demotedPlaceholder := env.createGroupRecord(t, "未分组", 1, false)

	require.NoError(t, RepairEmailGroupInvariantsForUser(ctx, env.db, env.user.ID))

	var reloadedPlaceholder models.EmailGroup
	require.NoError(t, env.db.First(&reloadedPlaceholder, demotedPlaceholder.ID).Error)
	require.NotNil(t, reloadedPlaceholder.SystemKey)
	require.Equal(t, models.EmailGroupSystemKeyDefaultPlaceholder, *reloadedPlaceholder.SystemKey)
	require.False(t, reloadedPlaceholder.IsDefault)

	var reloadedDefault models.EmailGroup
	require.NoError(t, env.db.First(&reloadedDefault, currentDefault.ID).Error)
	require.True(t, reloadedDefault.IsDefault)
	require.Nil(t, reloadedDefault.SystemKey)
}

func TestRepairEmailGroupInvariantsForUserMovesAccountsOffHiddenSystemGroup(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	defaultGroup := env.ensureDefaultGroup(t)
	hiddenSystemGroup := env.createSystemGroupRecord(t, "旧占位", 1, false, "legacy_placeholder")
	account := env.createAccountRecord(t, "hidden-system@qq.com", &hiddenSystemGroup.ID)

	require.NoError(t, RepairEmailGroupInvariantsForUser(ctx, env.db, env.user.ID))

	reloadedAccount := reloadAccount(t, env.db, account.ID)
	require.NotNil(t, reloadedAccount.GroupID)
	require.Equal(t, defaultGroup.ID, *reloadedAccount.GroupID)
}

func TestValidateEmailGroupInvariantsForUserRejectsHiddenSystemGroupAssignments(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	_ = env.ensureDefaultGroup(t)
	hiddenSystemGroup := env.createSystemGroupRecord(t, "旧占位", 1, false, "legacy_placeholder")
	_ = env.createAccountRecord(t, "hidden-system@qq.com", &hiddenSystemGroup.ID)

	err := ValidateEmailGroupInvariantsForUser(ctx, env.db, env.user.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrEmailGroupInvariantViolation))
	require.ErrorContains(t, err, "隐藏系统分组")
}

func TestGetEmailAccountsAndGroupsRemainPureReadWhenInvariantBroken(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	defaultGroup := env.createGroupRecord(t, "未分组", 0, true)
	account := env.createAccountRecord(t, "broken@qq.com", nil)

	_, err := env.service.GetEmailAccounts(ctx, env.user.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrEmailGroupInvariantViolation))

	reloadedAccount := reloadAccount(t, env.db, account.ID)
	require.Nil(t, reloadedAccount.GroupID)

	var reloadedDefault models.EmailGroup
	require.NoError(t, env.db.First(&reloadedDefault, defaultGroup.ID).Error)
	require.Nil(t, reloadedDefault.SystemKey)

	_, err = env.service.GetEmailGroups(ctx, env.user.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrEmailGroupInvariantViolation))

	reloadedAccount = reloadAccount(t, env.db, account.ID)
	require.Nil(t, reloadedAccount.GroupID)
}

func TestSystemGroupRestrictions(t *testing.T) {
	env := setupEmailGroupServiceTestEnv(t)
	ctx := context.Background()

	defaultGroup := env.ensureDefaultGroup(t)
	require.NotNil(t, defaultGroup.SystemKey)
	require.Equal(t, models.EmailGroupSystemKeyDefaultPlaceholder, *defaultGroup.SystemKey)

	hiddenPlaceholder := env.createSystemGroupRecord(t, "旧占位", 1, false, "legacy_placeholder")
	workGroup := env.createGroupRecord(t, "工作", 2, false)
	account := env.createAccountRecord(t, "system-guard@qq.com", &workGroup.ID)

	_, err := env.service.UpdateEmailGroup(ctx, env.user.ID, hiddenPlaceholder.ID, &UpdateEmailGroupRequest{Name: ptrString("新名字")})
	require.ErrorContains(t, err, "系统分组不可编辑")

	err = env.service.DeleteEmailGroup(ctx, env.user.ID, hiddenPlaceholder.ID)
	require.ErrorContains(t, err, "系统分组不可删除")

	_, err = env.service.ResolveEmailGroup(ctx, env.user.ID, &hiddenPlaceholder.ID)
	require.ErrorContains(t, err, "系统占位分组不可直接分配邮箱")

	err = env.service.MoveAccountToGroup(ctx, env.user.ID, account.ID, &hiddenPlaceholder.ID)
	require.ErrorContains(t, err, "系统占位分组不可直接分配邮箱")

	_, err = env.service.SetDefaultEmailGroup(ctx, env.user.ID, hiddenPlaceholder.ID)
	require.ErrorContains(t, err, "系统占位分组不可设为默认分组")

	_, err = env.service.ReorderEmailGroups(ctx, env.user.ID, []uint{hiddenPlaceholder.ID, workGroup.ID})
	require.ErrorContains(t, err, "系统分组不可参与排序")
}

func ptrString(value string) *string {
	return &value
}
