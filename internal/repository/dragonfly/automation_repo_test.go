package dragonfly

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/atm-ucak/follup/internal/domain"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestFollowupRepository(t *testing.T) (*FollowupRepository, *miniredis.Miniredis) {
	t.Helper()

	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	repo := NewFollowupRepository(client)

	return repo, s
}

func TestFollowupRepo_CreateAndGet_Success(t *testing.T) {
	repo, s := setupTestFollowupRepository(t)
	defer s.Close()

	ctx := context.Background()

	rule := &domain.Followup{
		UserID:        "user123",
		JiraTicketID:  "ticket456",
		JiraTicketKey: "PROJ-123",
		To:            "test@example.com",
		Subject:       "Follow-up: PROJ-123",
		EmailBody:     "Please review.",
		Frequency:     "0 9 * * 1",
		Status:        domain.FollowupStatusOngoing,
		CreatedAt:     time.Now(),
	}

	err := repo.Create(ctx, rule)
	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID, "ID should be generated")

	retrieved, err := repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)

	assert.Equal(t, rule.ID, retrieved.ID)
	assert.Equal(t, rule.UserID, retrieved.UserID)
	assert.Equal(t, rule.JiraTicketID, retrieved.JiraTicketID)
	assert.Equal(t, rule.Status, retrieved.Status)
}

func TestFollowupRepo_GetByUser_Success(t *testing.T) {
	repo, s := setupTestFollowupRepository(t)
	defer s.Close()

	ctx := context.Background()
	userID := "user123"

	rule1 := &domain.Followup{
		UserID:        userID,
		JiraTicketID:  "ticket1",
		JiraTicketKey: "PROJ-1",
		To:            "test1@example.com",
		Subject:       "Subject 1",
		EmailBody:     "Body 1",
		Frequency:     "0 9 * * 1",
		Status:        domain.FollowupStatusOngoing,
	}

	rule2 := &domain.Followup{
		UserID:        userID,
		JiraTicketID:  "ticket2",
		JiraTicketKey: "PROJ-2",
		To:            "test2@example.com",
		Subject:       "Subject 2",
		EmailBody:     "Body 2",
		Frequency:     "0 10 * * 2",
		Status:        domain.FollowupStatusStopped,
	}

	err := repo.Create(ctx, rule1)
	require.NoError(t, err)

	err = repo.Create(ctx, rule2)
	require.NoError(t, err)

	rules, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, rules, 2)
}

func TestFollowupRepo_GetActiveRules_Success(t *testing.T) {
	repo, s := setupTestFollowupRepository(t)
	defer s.Close()

	ctx := context.Background()

	active1 := &domain.Followup{
		UserID:        "user1",
		JiraTicketID:  "t1",
		JiraTicketKey: "PROJ-1",
		To:            "a@example.com",
		Subject:       "S1",
		EmailBody:     "B1",
		Frequency:     "0 9 * * 1",
		Status:        domain.FollowupStatusOngoing,
	}
	active2 := &domain.Followup{
		UserID:        "user2",
		JiraTicketID:  "t2",
		JiraTicketKey: "PROJ-2",
		To:            "b@example.com",
		Subject:       "S2",
		EmailBody:     "B2",
		Frequency:     "0 10 * * 2",
		Status:        domain.FollowupStatusOngoing,
	}
	paused := &domain.Followup{
		UserID:        "user1",
		JiraTicketID:  "t3",
		JiraTicketKey: "PROJ-3",
		To:            "c@example.com",
		Subject:       "S3",
		EmailBody:     "B3",
		Frequency:     "0 11 * * 3",
		Status:        domain.FollowupStatusStopped,
	}

	require.NoError(t, repo.Create(ctx, active1))
	require.NoError(t, repo.Create(ctx, active2))
	require.NoError(t, repo.Create(ctx, paused))

	rules, err := repo.GetActiveRules(ctx)
	require.NoError(t, err)
	assert.Len(t, rules, 2)
}

func TestFollowupRepo_Update_Success(t *testing.T) {
	repo, s := setupTestFollowupRepository(t)
	defer s.Close()

	ctx := context.Background()

	rule := &domain.Followup{
		UserID:        "user123",
		JiraTicketID:  "ticket456",
		JiraTicketKey: "PROJ-123",
		To:            "test@example.com",
		Subject:       "Original Subject",
		EmailBody:     "Original body",
		Frequency:     "0 9 * * 1",
		Status:        domain.FollowupStatusOngoing,
	}

	err := repo.Create(ctx, rule)
	require.NoError(t, err)

	rule.To = "updated@example.com"
	rule.Frequency = "0 10 * * 2"
	rule.Subject = "Updated Subject"
	err = repo.Update(ctx, rule)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated@example.com", retrieved.To)
	assert.Equal(t, "0 10 * * 2", retrieved.Frequency)
	assert.Equal(t, "Updated Subject", retrieved.Subject)
}

func TestFollowupRepo_Delete_Success(t *testing.T) {
	repo, s := setupTestFollowupRepository(t)
	defer s.Close()

	ctx := context.Background()

	rule := &domain.Followup{
		UserID:        "user123",
		JiraTicketID:  "ticket456",
		JiraTicketKey: "PROJ-123",
		To:            "test@example.com",
		Subject:       "S",
		EmailBody:     "B",
		Frequency:     "0 9 * * 1",
		Status:        domain.FollowupStatusOngoing,
	}

	err := repo.Create(ctx, rule)
	require.NoError(t, err)

	err = repo.Delete(ctx, rule.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, rule.ID)
	assert.Error(t, err)
}
