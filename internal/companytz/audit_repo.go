package companytz

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepoAudit struct {
	pg *pgxpool.Pool
}

func NewRepoAudit(pg *pgxpool.Pool) *RepoAudit {
	return &RepoAudit{pg: pg}
}

func (r *RepoAudit) Log(ctx context.Context, userID, companyID *uuid.UUID, action, entityType string, entityID uuid.UUID, oldData, newData interface{}) error {
	var oldJSON, newJSON []byte
	if oldData != nil {
		oldJSON, _ = json.Marshal(oldData)
	}
	if newData != nil {
		newJSON, _ = json.Marshal(newData)
	}
	_, err := r.pg.Exec(ctx, `
INSERT INTO audit_log (user_id, company_id, action, entity_type, entity_id, old_data, new_data)
VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, companyID, action, entityType, entityID, oldJSON, newJSON)
	return err
}
