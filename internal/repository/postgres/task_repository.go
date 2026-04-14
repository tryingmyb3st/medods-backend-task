package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const queryTasks = `
		INSERT INTO tasks (title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, status, created_at, updated_at
	`

	const queryPeriodicity = `
	INSERT INTO periodicity (task_id, period_type, days_interval, day_of_month, custom_dates, even_odd)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING period_type, days_interval, day_of_month, custom_dates, even_odd, last_created_at
	`

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, queryTasks, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt)
	createdTask, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	var createdPeriodicity *taskdomain.Periodicity
	if task.Schedule != nil {
		row := tx.QueryRow(
			ctx,
			queryPeriodicity,
			createdTask.ID,
			task.Schedule.Type,
			task.Schedule.DaysInterval,
			task.Schedule.DayOfMonth,
			task.Schedule.CustomDates,
			task.Schedule.EvenOdd,
		)

		createdPeriodicity, err = ScanPeriodicity(row)
		if err != nil {
			return nil, err
		}
	}

	createdTask.Schedule = createdPeriodicity

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return createdTask, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at,
					period_type, days_interval, day_of_month, custom_dates, even_odd, last_created_at
		FROM tasks
		LEFT JOIN periodicity ON tasks.id = periodicity.task_id
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := ScanTaskWithPeriodicity(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const queryTasks = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			updated_at = $4
		WHERE id = $5
		RETURNING id, title, description, status, created_at, updated_at
	`

	const queryPeriodicity = `
	INSERT INTO periodicity (task_id, period_type, days_interval, day_of_month, custom_dates, even_odd)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (task_id) DO UPDATE SET
		period_type = EXCLUDED.period_type,
		days_interval = EXCLUDED.days_interval,
		day_of_month = EXCLUDED.day_of_month,
		custom_dates = EXCLUDED.custom_dates,
		even_odd = EXCLUDED.even_odd
	RETURNING period_type, days_interval, day_of_month, custom_dates, even_odd, last_created_at
	`

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, queryTasks, task.Title, task.Description, task.Status, task.UpdatedAt, task.ID)
	updatedTask, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	var updatedPeriodicity *taskdomain.Periodicity
	if task.Schedule != nil {
		row := tx.QueryRow(
			ctx,
			queryPeriodicity,
			updatedTask.ID,
			task.Schedule.Type,
			task.Schedule.DaysInterval,
			task.Schedule.DayOfMonth,
			task.Schedule.CustomDates,
			task.Schedule.EvenOdd,
		)

		updatedPeriodicity, err = ScanPeriodicity(row)
		if err != nil {
			return nil, err
		}
	}

	if task.Schedule == nil {
		queryDelete := `
		DELETE FROM periodicity
		WHERE task_id = $1
		`
		_, err := tx.Exec(ctx, queryDelete, updatedTask.ID)
		if err != nil {
			return nil, err
		}
	}

	updatedTask.Schedule = updatedPeriodicity

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return updatedTask, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at,
					period_type, days_interval, day_of_month, custom_dates, even_odd, last_created_at
		FROM tasks
		LEFT JOIN periodicity ON tasks.id = periodicity.task_id
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := ScanTaskWithPeriodicity(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) GetTemplates(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
	SELECT id, title, description, status, created_at, updated_at,
				period_type, days_interval, day_of_month, custom_dates, even_odd, last_created_at
	FROM tasks
	JOIN periodicity ON tasks.id = periodicity.task_id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := ScanTaskWithPeriodicity(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *Repository) UpdateLastCreatedAt(ctx context.Context, taskIdd int64, updatedTime time.Time) error {
	query := `
	UPDATE periodicity
	SET last_created_at=$1
	WHERE task_id=$2
	`

	_, err := s.pool.Exec(ctx, query, updatedTime, taskIdd)
	return err
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task   taskdomain.Task
		status string
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	return &task, nil
}

func ScanPeriodicity(scanner taskScanner) (*taskdomain.Periodicity, error) {
	var (
		periodicity taskdomain.Periodicity
		periodType  taskdomain.PeriodType
	)

	if err := scanner.Scan(
		&periodType,
		&periodicity.DaysInterval,
		&periodicity.DayOfMonth,
		&periodicity.CustomDates,
		&periodicity.EvenOdd,
		&periodicity.LastCreatedAt,
	); err != nil {
		return nil, err
	}

	periodicity.Type = taskdomain.PeriodType(periodType)

	return &periodicity, nil
}

func ScanTaskWithPeriodicity(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task       taskdomain.Task
		schedule   taskdomain.Periodicity
		status     string
		PeriodType sql.NullString
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
		&PeriodType,
		&schedule.DaysInterval,
		&schedule.DayOfMonth,
		&schedule.CustomDates,
		&schedule.EvenOdd,
		&schedule.LastCreatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	if PeriodType.Valid {
		schedule.Type = taskdomain.PeriodType(PeriodType.String)
		task.Schedule = &schedule
	}

	return &task, nil
}
