package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

type LifecycleProcessor struct {
	store  *dbgen.Queries
	sender Sender
	now    func() time.Time
}

type outingJob struct {
	JourneyID uuid.UUID `json:"journey_id"`
}

func NewLifecycleProcessor(store *dbgen.Queries, sender Sender) *LifecycleProcessor {
	return &LifecycleProcessor{store: store, sender: sender, now: time.Now}
}

func (p *LifecycleProcessor) HandleWarning(ctx context.Context, payload []byte) error {
	job, err := decodeOutingJob(payload)
	if err != nil {
		return err
	}
	journey, err := p.store.GetJourneyByID(ctx, pgUUID(job.JourneyID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get outing for warning: %w", err)
	}
	if journey.Kind != "outing" || journey.Status != "active" {
		return nil
	}

	subscriptions, err := p.store.ListPushSubscriptionsForUser(ctx, journey.OwnerID)
	if err != nil {
		return fmt.Errorf("list push subscriptions: %w", err)
	}
	notification, _ := json.Marshal(map[string]any{
		"title": "Is your outing now a trip?",
		"body":  "This outing ends automatically in one hour. Convert it to a trip to keep recording.",
		"data":  map[string]string{"journey_id": job.JourneyID.String(), "action": "convert-to-trip"},
	})
	for _, subscription := range subscriptions {
		err := p.sender.Send(ctx, subscription.PushSubscription, notification)
		if errors.Is(err, ErrSubscriptionExpired) {
			if clearErr := p.store.ClearDevicePushSubscription(ctx, subscription.ID); clearErr != nil {
				return fmt.Errorf("clear expired push subscription: %w", clearErr)
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *LifecycleProcessor) HandleExpiry(ctx context.Context, payload []byte) error {
	job, err := decodeOutingJob(payload)
	if err != nil {
		return err
	}
	journey, err := p.store.GetJourneyByID(ctx, pgUUID(job.JourneyID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get outing for expiry: %w", err)
	}
	if journey.Kind != "outing" || journey.Status != "active" {
		return nil
	}
	_, err = p.store.ExpireJourney(ctx, dbgen.ExpireJourneyParams{
		ID: pgUUID(job.JourneyID), EndedAt: pgtype.Timestamptz{Time: p.now().UTC(), Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("expire outing: %w", err)
	}
	return nil
}

func decodeOutingJob(payload []byte) (outingJob, error) {
	var job outingJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return outingJob{}, fmt.Errorf("decode outing lifecycle job: %w", err)
	}
	if job.JourneyID == uuid.Nil {
		return outingJob{}, fmt.Errorf("decode outing lifecycle job: journey ID is required")
	}
	return job, nil
}

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}
