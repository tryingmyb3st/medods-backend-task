package scheduler

import (
	"context"
	"log/slog"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type PeriodicityScheduler struct {
	usecase       taskusecase.Usecase
	refreshTicker *time.Ticker
	logger        *slog.Logger
}

func NewPeriodicityScheduler(usecase taskusecase.Usecase, log *slog.Logger, refreshTime time.Duration) *PeriodicityScheduler {
	ticker := time.NewTicker(refreshTime)
	return &PeriodicityScheduler{
		usecase:       usecase,
		refreshTicker: ticker,
		logger:        log,
	}
}

func (s *PeriodicityScheduler) Close() {
	s.refreshTicker.Stop()
}

func (s *PeriodicityScheduler) Start(ctx context.Context) {
	go func() {
		defer s.Close()
		for {
			select {
			case <-ctx.Done():
				s.logger.Info("scheduler stop")
				return
			case <-s.refreshTicker.C:
				s.logger.Info("start processing templates...")
				if err := s.processTemplates(ctx); err != nil {
					s.logger.Info("err processing templates", "error", err)
				}
			}
		}
	}()
}

func (s *PeriodicityScheduler) processTemplates(ctx context.Context) error {
	templates, err := s.usecase.GetTemplates(ctx)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		return nil
	}

	now := time.Now()

	for _, task := range templates {
		switch task.Schedule.Type {
		case taskdomain.Daily:
			s.processDaily(ctx, &task, now)
		case taskdomain.Monthly:
			s.processMonthly(ctx, &task, now)
		case taskdomain.Custom:
			s.processDates(ctx, &task, now)
		case taskdomain.EvenOddDates:
			s.processEvenOdd(ctx, &task, now)
		}
	}

	return nil
}

func (s *PeriodicityScheduler) processDaily(ctx context.Context, task *taskdomain.Task, now time.Time) {
	diff := now.Sub(task.CreatedAt)

	daysSince := int64(diff.Hours() / 24)

	if daysSince == 0 || daysSince%(*task.Schedule.DaysInterval) != 0 {
		return
	}

	year, month, day := task.Schedule.LastCreatedAt.Date()

	if now.Year() == year && now.Month() == month && now.Day() == day {
		return
	}

	created, err := s.usecase.Create(ctx, taskusecase.CreateInput{
		Title:       task.Title,
		Description: task.Description,
		Status:      taskdomain.StatusNew,
	})
	if err != nil {
		s.logger.Error("creating new task by daily schedule", slog.Any("err", err))
		return
	}

	err = s.usecase.UpdateLastCreatedAt(ctx, task.ID, now)
	if err != nil {
		s.logger.Error("updating last created at", slog.Any("err", err))
		return
	}
	s.logger.Debug("created new task by daily schedule", slog.Any("id", created.ID))
}

func (s *PeriodicityScheduler) processMonthly(ctx context.Context, task *taskdomain.Task, now time.Time) {
	today := int64(now.Day())

	if *task.Schedule.DayOfMonth != today {
		return
	}

	year, month, day := task.Schedule.LastCreatedAt.Date()
	if now.Year() == year && now.Month() == month && now.Day() == day {
		return
	}

	created, err := s.usecase.Create(ctx, taskusecase.CreateInput{
		Title:       task.Title,
		Description: task.Description,
		Status:      taskdomain.StatusNew,
	})

	if err != nil {
		s.logger.Error("creating new task by monthly schedule", slog.Any("err", err))
		return
	}

	err = s.usecase.UpdateLastCreatedAt(ctx, task.ID, now)
	if err != nil {
		s.logger.Error("updating last created at", slog.Any("err", err))
		return
	}
	s.logger.Debug("created new task by monthly schedule", slog.Any("id", created.ID))
}

func (s *PeriodicityScheduler) processDates(ctx context.Context, task *taskdomain.Task, now time.Time) {
	for _, date := range task.Schedule.CustomDates {
		if !(now.Day() == date.Day() && now.Month() == date.Month() && now.Year() == date.Year()) {
			continue
		}

		year, month, day := task.Schedule.LastCreatedAt.Date()
		if now.Year() == year && now.Month() == month && now.Day() == day {
			continue
		}

		created, err := s.usecase.Create(ctx, taskusecase.CreateInput{
			Title:       task.Title,
			Description: task.Description,
			Status:      taskdomain.StatusNew,
		})

		if err != nil {
			s.logger.Error("creating new task by date schedule", slog.Any("err", err))
			continue
		}

		err = s.usecase.UpdateLastCreatedAt(ctx, task.ID, now)
		if err != nil {
			s.logger.Error("updating last created at", slog.Any("err", err))
			continue
		}
		s.logger.Debug("created new task by date schedule", slog.Any("id", created.ID))
	}
}

func (s *PeriodicityScheduler) processEvenOdd(ctx context.Context, task *taskdomain.Task, now time.Time) {
	today := int64(now.Day())

	if !((*task.Schedule.EvenOdd == "even" && today%2 == 0) ||
		(*task.Schedule.EvenOdd == "odd" && today%2 == 1)) {
		return
	}

	year, month, day := task.Schedule.LastCreatedAt.Date()
	if now.Year() == year && now.Month() == month && now.Day() == day {
		return
	}

	created, err := s.usecase.Create(ctx, taskusecase.CreateInput{
		Title:       task.Title,
		Description: task.Description,
		Status:      taskdomain.StatusNew,
	})

	if err != nil {
		s.logger.Error("creating new task by even odd schedule", slog.Any("err", err))
		return
	}

	err = s.usecase.UpdateLastCreatedAt(ctx, task.ID, now)
	if err != nil {
		s.logger.Error("updating last created at", slog.Any("err", err))
		return
	}
	s.logger.Debug("created new task by even odd schedule", slog.Any("id", created.ID))
}
