package service

import (
	"Test/internal/model"
	"Test/internal/repository"
	"context"
	"fmt"
)

type UserService struct {
	userRepo repository.UserRepository
	taskRepo repository.TaskRepository
}

func NewUserService(userRepo repository.UserRepository, taskRepo repository.TaskRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
		taskRepo: taskRepo,
	}
}

func (s *UserService) GetUserStatus(ctx context.Context, userID string) (*model.UserStatus, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	tasks, err := s.taskRepo.GetCompletedTasks(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed tasks: %w", err)
	}

	var referrals []model.User
	if user.Referrer != nil {
		referrals, err = s.userRepo.GetReferrals(ctx, *user.Referrer)
		if err != nil {
			return nil, fmt.Errorf("failed to get referrals: %w", err)
		}
	}

	return &model.UserStatus{
		User:           *user,
		CompletedTasks: tasks,
		Referrals:      referrals,
	}, nil
}

func (s *UserService) CompleteTask(ctx context.Context, userID, taskID string) error {
	return s.taskRepo.CompleteTask(ctx, userID, taskID)
}

func (s *UserService) SetReferrer(ctx context.Context, userID, referrerID string) error {
	if _, err := s.userRepo.GetUserByID(ctx, referrerID); err != nil {
		return err
	}

	if err := s.userRepo.UpdateUserPoints(ctx, referrerID, 100); err != nil {
		return err
	}

	return s.userRepo.SetReferrer(ctx, userID, referrerID)
}

func (s *UserService) GetLeaderboard(ctx context.Context, limit int) ([]model.LeaderboardEntry, error) {
	return s.userRepo.GetLeaderboard(ctx, limit)
}
