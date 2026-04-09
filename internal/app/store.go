package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Register the pure-Go SQLite driver used for plan/job state.
)

type Store struct {
	db *sql.DB
}

func NewStore(stateDir string) (*Store, error) {
	if err := os.MkdirAll(stateDir, 0o750); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(stateDir, "state.db"))
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	stmts := []string{
		`create table if not exists plans (
			plan_id text primary key,
			created_at text not null,
			expires_at text not null,
			kube_context text not null,
			linstor_cluster text not null,
			kind text not null,
			name text not null,
			operation text not null,
			desired_spec text not null,
			summary text not null,
			diff text not null,
			destructive integer not null,
			preconditions text not null,
			state text not null
		)`,
		`create table if not exists jobs (
			job_id text primary key,
			plan_id text not null,
			idempotency_key text not null,
			phase text not null,
			result_ref text not null default '',
			error text not null default '',
			created_at text not null,
			updated_at text not null,
			unique(plan_id, idempotency_key)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SavePlan(ctx context.Context, plan PlanRecord) error {
	preconditions, err := json.Marshal(plan.Preconditions)
	if err != nil {
		return err
	}
	spec, err := json.Marshal(plan.DesiredSpec)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		insert into plans(plan_id, created_at, expires_at, kube_context, linstor_cluster, kind, name, operation, desired_spec, summary, diff, destructive, preconditions, state)
		values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, plan.PlanID, plan.CreatedAt.Format(time.RFC3339Nano), plan.ExpiresAt.Format(time.RFC3339Nano), plan.KubeContext, plan.LinstorCluster, string(plan.Kind), plan.Name, plan.Operation, string(spec), plan.Summary, plan.Diff, boolToInt(plan.Destructive), string(preconditions), plan.State)
	return err
}

func (s *Store) GetPlan(ctx context.Context, planID string) (*PlanRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		select plan_id, created_at, expires_at, kube_context, linstor_cluster, kind, name, operation, desired_spec, summary, diff, destructive, preconditions, state
		from plans where plan_id = ?
	`, planID)
	var (
		plan          PlanRecord
		kind          string
		spec          string
		preconditions string
		createdAt     string
		expiresAt     string
		destructive   int
	)
	if err := row.Scan(&plan.PlanID, &createdAt, &expiresAt, &plan.KubeContext, &plan.LinstorCluster, &kind, &plan.Name, &plan.Operation, &spec, &plan.Summary, &plan.Diff, &destructive, &preconditions, &plan.State); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	plan.Kind = InventoryKind(kind)
	plan.Destructive = destructive == 1
	if err := json.Unmarshal([]byte(spec), &plan.DesiredSpec); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(preconditions), &plan.Preconditions); err != nil {
		return nil, err
	}
	var err error
	plan.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, err
	}
	plan.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (s *Store) GetOrCreateJob(ctx context.Context, planID, idempotencyKey string) (*JobRecord, bool, error) {
	job, err := s.findJob(ctx, planID, idempotencyKey)
	if err != nil {
		return nil, false, err
	}
	if job != nil {
		return job, true, nil
	}
	now := time.Now().UTC()
	job = &JobRecord{
		JobID:          newID("job"),
		PlanID:         planID,
		IdempotencyKey: idempotencyKey,
		Phase:          "running",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	_, err = s.db.ExecContext(ctx, `
		insert into jobs(job_id, plan_id, idempotency_key, phase, result_ref, error, created_at, updated_at)
		values(?, ?, ?, ?, ?, ?, ?, ?)
	`, job.JobID, job.PlanID, job.IdempotencyKey, job.Phase, job.ResultRef, job.Error, job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, false, err
	}
	return job, false, nil
}

func (s *Store) findJob(ctx context.Context, planID, idempotencyKey string) (*JobRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		select job_id, plan_id, idempotency_key, phase, result_ref, error, created_at, updated_at
		from jobs where plan_id = ? and idempotency_key = ?
	`, planID, idempotencyKey)
	return scanJob(row)
}

func (s *Store) GetJob(ctx context.Context, jobID string) (*JobRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		select job_id, plan_id, idempotency_key, phase, result_ref, error, created_at, updated_at
		from jobs where job_id = ?
	`, jobID)
	return scanJob(row)
}

func scanJob(row *sql.Row) (*JobRecord, error) {
	var (
		job       JobRecord
		createdAt string
		updatedAt string
	)
	if err := row.Scan(&job.JobID, &job.PlanID, &job.IdempotencyKey, &job.Phase, &job.ResultRef, &job.Error, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var err error
	job.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, err
	}
	job.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *Store) UpdateJob(ctx context.Context, job JobRecord) error {
	job.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		update jobs set phase = ?, result_ref = ?, error = ?, updated_at = ? where job_id = ?
	`, job.Phase, job.ResultRef, job.Error, job.UpdatedAt.Format(time.RFC3339Nano), job.JobID)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
}
