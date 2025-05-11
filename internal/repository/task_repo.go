package repository

import (
	"Test/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type TaskRepo struct {
	db *sql.DB
}

type TaskRepository interface {
	GetTaskByID(ctx context.Context, id string) (*model.Task, error)
	GetCompletedTasks(ctx context.Context, userID string) ([]model.Task, error)
	CompleteTask(ctx context.Context, userID, taskID string) error
	GetReferrals(ctx context.Context, userID string) ([]model.User, error)
}

func NewTaskRepo(db *sql.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) GetTaskByID(ctx context.Context, id string) (*model.Task, error) {
	query := `SELECT id, name, description, points FROM tasks WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var task model.Task
	if err := row.Scan(&task.ID, &task.Name, &task.Description, &task.Points); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return &task, nil
}

func (r *TaskRepo) GetCompletedTasks(ctx context.Context, userID string) ([]model.Task, error) {
	query := `SELECT t.id, t.name, t.description, t.points 
		FROM tasks t
		JOIN user_tasks ut ON t.id = ut.task_id
		WHERE ut.user_id = $1 AND ut.completed_at IS NOT NULL`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query completed tasks: %w", err)
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var task model.Task
		if err := rows.Scan(&task.ID, &task.Name, &task.Description, &task.Points); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepo) CompleteTask(ctx context.Context, userID, taskName string) error {
	var taskID string
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM tasks WHERE name = $1`, taskName).Scan(&taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	task, err := r.GetTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task check failed: %w", err)
	}

	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM user_tasks WHERE user_id = $1 AND task_id = $2 AND completed_at IS NOT NULL)`
	err = r.db.QueryRowContext(ctx, checkQuery, userID, taskID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check task completion: %w", err)
	}

	if exists {
		return errors.New("task already completed")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO user_tasks (user_id, task_id, completed_at) 
         VALUES ($1, $2, $3)
         ON CONFLICT (user_id, task_id) DO UPDATE SET completed_at = $3`,
		userID, taskID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE users SET points = points + $1, updated_at = $2 WHERE id = $3`,
		task.Points, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user points: %w", err)
	}

	if taskID == "3" {
		var referrerID sql.NullString
		err = tx.QueryRowContext(ctx,
			`SELECT referrer FROM users WHERE id = $1`,
			userID).Scan(&referrerID)
		if err != nil {
			return fmt.Errorf("failed to get referrer: %w", err)
		}

		if referrerID.Valid && referrerID.String != "" {
			_, err = tx.ExecContext(ctx,
				`UPDATE users SET points = points + $1, updated_at = $2 WHERE id = $3`,
				task.Points/2, time.Now(), referrerID.String)
			if err != nil {
				return fmt.Errorf("failed to update referrer points: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *TaskRepo) GetReferrals(ctx context.Context, userID string) ([]model.User, error) {
	query := `
		SELECT u.id, u.name, u.email, u.points, u.created_at, u.updated_at
		FROM users u
		JOIN referrals r ON u.id = r.referee_id
		WHERE r.referrer_id = $1
		ORDER BY r.date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query referrals: %w", err)
	}
	defer rows.Close()

	var referrals []model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Points,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan referral: %w", err)
		}
		referrals = append(referrals, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return referrals, nil
}
