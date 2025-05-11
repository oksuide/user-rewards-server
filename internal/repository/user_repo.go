package repository

import (
	"Test/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type UserRepo struct {
	db *sql.DB
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateUserPoints(ctx context.Context, id string, points int) error
	SetReferrer(ctx context.Context, id, referrerID string) error
	GetReferrals(ctx context.Context, referrerID string) ([]model.User, error)
	GetLeaderboard(ctx context.Context, limit int) ([]model.LeaderboardEntry, error)
	EmailExists(ctx context.Context, email string) (bool, error)
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (id, name, email, password, points, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Name, user.Email, user.Password, user.Points, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT id, name, email, password, points, referrer, created_at, updated_at 
              FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var user model.User
	var referrer sql.NullString
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Points,
		&referrer, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	if referrer.Valid {
		user.Referrer = &referrer.String
	}

	return &user, nil
}

func (r *UserRepo) UpdateUserPoints(ctx context.Context, id string, points int) error {
	query := `UPDATE users SET points = points + $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, points, id)
	if err != nil {
		return fmt.Errorf("failed to update user points: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (r *UserRepo) SetReferrer(ctx context.Context, userId, referrerId string) error {
	if userId == referrerId {
		return errors.New("user cannot be their own referrer")
	}

	var userExists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`,
		userId).Scan(&userExists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if !userExists {
		return errors.New("user not found")
	}

	var referrerExists bool
	err = r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`,
		referrerId).Scan(&referrerExists)
	if err != nil {
		return fmt.Errorf("failed to check referrer existence: %w", err)
	}
	if !referrerExists {
		return errors.New("referrer not found")
	}

	var hasReferrer bool
	err = r.db.QueryRowContext(ctx,
		`SELECT referrer IS NOT NULL FROM users WHERE id = $1`,
		userId).Scan(&hasReferrer)
	if err != nil {
		return fmt.Errorf("failed to check referrer status: %w", err)
	}
	if hasReferrer {
		return errors.New("referrer already set")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`UPDATE users SET referrer = $1, updated_at = $2 WHERE id = $3`,
		referrerId, time.Now(), userId)
	if err != nil {
		return fmt.Errorf("failed to set referrer: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE users SET points = points + 100, updated_at = $1 WHERE id = $2`,
		time.Now(), referrerId)
	if err != nil {
		return fmt.Errorf("failed to add referral bonus: %w", err)
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO user_tasks (user_id, task_id, completed_at)
         VALUES ($1, '3', $2)  -- '3' это ID реферального задания
         ON CONFLICT (user_id, task_id) DO UPDATE SET completed_at = $2`,
		userId, time.Now())
	if err != nil {
		return fmt.Errorf("failed to complete referral task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *UserRepo) GetLeaderboard(ctx context.Context, limit int) ([]model.LeaderboardEntry, error) {
	query := `
        SELECT 
            id, 
            name, 
            points,
            ROW_NUMBER() OVER (ORDER BY points DESC) as position
        FROM users 
        ORDER BY points DESC 
        LIMIT $1
    `

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var leaderboard []model.LeaderboardEntry
	for rows.Next() {
		var entry model.LeaderboardEntry
		if err := rows.Scan(
			&entry.UserID,
			&entry.Name,
			&entry.Points,
			&entry.Position,
		); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		leaderboard = append(leaderboard, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return leaderboard, nil
}

func (r *UserRepo) GetReferrals(ctx context.Context, referrerID string) ([]model.User, error) {
	query := `
        SELECT u.id, u.name, u.email, u.password, u.points, u.referrer, u.created_at, u.updated_at
        FROM users u
        JOIN referrals r ON u.id = r.referee_id
        WHERE r.referrer_id = $1
    `
	rows, err := r.db.QueryContext(ctx, query, referrerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query referrals: %w", err)
	}
	defer rows.Close()

	var referrals []model.User
	for rows.Next() {
		var user model.User
		var referrer sql.NullString
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Points, &referrer, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan referral user: %w", err)
		}
		if referrer.Valid {
			user.Referrer = &referrer.String
		}
		referrals = append(referrals, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return referrals, nil
}

func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, name, email, password, points, referrer, created_at, updated_at 
              FROM users WHERE email = $1`
	row := r.db.QueryRowContext(ctx, query, email)

	var user model.User
	var referrer sql.NullString
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Points,
		&referrer, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if referrer.Valid {
		user.Referrer = &referrer.String
	}

	return &user, nil
}

func (r *UserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return exists, nil
}
