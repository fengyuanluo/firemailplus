package services

import (
	"context"
	"fmt"
	"testing"

	"firemail/internal/models"
	"firemail/internal/providers"
	"firemail/internal/sse"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type recordingEventPublisher struct {
	events []*sse.Event
}

func (p *recordingEventPublisher) Publish(_ context.Context, event *sse.Event) error {
	if event != nil {
		p.events = append(p.events, event)
	}
	return nil
}

func (p *recordingEventPublisher) PublishToUser(ctx context.Context, userID uint, event *sse.Event) error {
	if event != nil {
		event.UserID = userID
	}
	return p.Publish(ctx, event)
}

func (p *recordingEventPublisher) PublishToAccount(ctx context.Context, _ uint, event *sse.Event) error {
	return p.Publish(ctx, event)
}

func (p *recordingEventPublisher) Broadcast(ctx context.Context, event *sse.Event) error {
	return p.Publish(ctx, event)
}

type emailGroupEventServiceTestEnv struct {
	*emailGroupServiceTestEnv
	publisher *recordingEventPublisher
}

func setupEmailGroupEventServiceTestEnv(t *testing.T) *emailGroupEventServiceTestEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.EmailGroup{}, &models.EmailAccount{}))

	user := &models.User{
		Username: fmt.Sprintf("event_user_%s", t.Name()),
		Password: "password123",
		Role:     "admin",
		IsActive: true,
	}
	require.NoError(t, db.Create(user).Error)

	publisher := &recordingEventPublisher{}
	service, ok := NewEmailService(db, providers.NewProviderFactory(), publisher).(*EmailServiceImpl)
	require.True(t, ok)

	return &emailGroupEventServiceTestEnv{
		emailGroupServiceTestEnv: &emailGroupServiceTestEnv{
			db:      db,
			service: service,
			user:    user,
		},
		publisher: publisher,
	}
}

func TestCreateEmailGroupPublishesGroupCreatedEvent(t *testing.T) {
	env := setupEmailGroupEventServiceTestEnv(t)
	ctx := context.Background()

	_ = env.ensureDefaultGroup(t)
	env.createGroupRecord(t, "现有分组", 1, false)

	created, err := env.service.CreateEmailGroup(ctx, env.user.ID, &CreateEmailGroupRequest{Name: "工作"})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEmpty(t, env.publisher.events)

	lastEvent := env.publisher.events[len(env.publisher.events)-1]
	require.Equal(t, sse.EventGroupCreated, lastEvent.Type)
	data, ok := lastEvent.Data.(*sse.GroupEventData)
	require.True(t, ok)
	require.Equal(t, created.ID, data.GroupID)
	require.Equal(t, created.Name, data.Name)
}

func TestMoveAccountToGroupPublishesAccountGroupChangedEvent(t *testing.T) {
	env := setupEmailGroupEventServiceTestEnv(t)
	ctx := context.Background()

	defaultGroup := env.ensureDefaultGroup(t)
	workGroup := env.createGroupRecord(t, "工作", 1, false)
	account := env.createAccountRecord(t, "move-event@qq.com", &workGroup.ID)

	require.NoError(t, env.service.MoveAccountToGroup(ctx, env.user.ID, account.ID, nil))
	require.NotEmpty(t, env.publisher.events)

	lastEvent := env.publisher.events[len(env.publisher.events)-1]
	require.Equal(t, sse.EventAccountGroupChanged, lastEvent.Type)
	data, ok := lastEvent.Data.(*sse.AccountGroupEventData)
	require.True(t, ok)
	require.Equal(t, account.ID, data.AccountID)
	require.NotNil(t, data.GroupID)
	require.Equal(t, defaultGroup.ID, *data.GroupID)
	require.NotNil(t, data.PreviousGroupID)
	require.Equal(t, workGroup.ID, *data.PreviousGroupID)
}

func TestSetDefaultEmailGroupPublishesDefaultChangedEvent(t *testing.T) {
	env := setupEmailGroupEventServiceTestEnv(t)
	ctx := context.Background()

	oldDefault := env.ensureDefaultGroup(t)
	newDefault := env.createGroupRecord(t, "工作", 1, false)

	updated, err := env.service.SetDefaultEmailGroup(ctx, env.user.ID, newDefault.ID)
	require.NoError(t, err)
	require.True(t, updated.IsDefault)
	require.NotEmpty(t, env.publisher.events)

	lastEvent := env.publisher.events[len(env.publisher.events)-1]
	require.Equal(t, sse.EventGroupDefaultChanged, lastEvent.Type)
	data, ok := lastEvent.Data.(*sse.GroupEventData)
	require.True(t, ok)
	require.Equal(t, newDefault.ID, data.GroupID)
	require.NotNil(t, data.PreviousDefaultGroupID)
	require.Equal(t, oldDefault.ID, *data.PreviousDefaultGroupID)
}
