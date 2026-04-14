package handlers

import (
	"encoding/json"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type CustomDate time.Time

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	Periodicity *PeriodicityDTO   `json:"periodicity"`
}

type PeriodicityDTO struct {
	Type         taskdomain.PeriodType `json:"period_type"`
	DaysInterval *int64                `json:"days_interval,omitempty"`
	DayOfMonth   *int64                `json:"day_of_month,omitempty"`
	CustomDates  []CustomDate          `json:"custom_dates,omitempty"`
	EvenOdd      *string               `json:"even_odd,omitempty"`
}

func (p *PeriodicityDTO) UnmarshalJSON(data []byte) error {
	type Alias PeriodicityDTO
	temp := Alias{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	periodType := taskdomain.PeriodType(temp.Type)
	switch periodType {
	case taskdomain.Daily:
		p.Type = taskdomain.Daily
		p.DaysInterval = temp.DaysInterval
	case taskdomain.Monthly:
		p.Type = taskdomain.Monthly
		p.DayOfMonth = temp.DayOfMonth
	case taskdomain.Custom:
		p.Type = taskdomain.Custom
		p.CustomDates = temp.CustomDates
	case taskdomain.EvenOddDates:
		p.Type = taskdomain.EvenOddDates
		p.EvenOdd = temp.EvenOdd
	}
	return nil
}

func (p *PeriodicityDTO) ToDomain() *taskdomain.Periodicity {
	if p == nil {
		return nil
	}

	dates := make([]time.Time, len(p.CustomDates))
	for i, d := range p.CustomDates {
		dates[i] = time.Time(d)
	}

	if len(p.CustomDates) == 0 {
		dates = nil
	}

	return &taskdomain.Periodicity{
		Type:         p.Type,
		DaysInterval: p.DaysInterval,
		DayOfMonth:   p.DayOfMonth,
		CustomDates:  dates,
		EvenOdd:      p.EvenOdd,
	}
}

type taskDTO struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Periodicity *PeriodicityDTO   `json:"periodicity,omitempty"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	dto := taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}

	if task.Schedule != nil {
		dates := make([]CustomDate, len(task.Schedule.CustomDates))
		for i, d := range task.Schedule.CustomDates {
			dates[i] = CustomDate(d)
		}

		dto.Periodicity = &PeriodicityDTO{
			Type:         task.Schedule.Type,
			DaysInterval: task.Schedule.DaysInterval,
			DayOfMonth:   task.Schedule.DayOfMonth,
			CustomDates:  dates,
			EvenOdd:      task.Schedule.EvenOdd,
		}
	}
	return dto
}

func (d *CustomDate) UnmarshalJSON(b []byte) error {
	layout := "2006-01-02"
	s := strings.Trim(string(b), "\"")
	date, err := time.ParseInLocation(layout, s, time.Local)
	if err != nil {
		return err
	}

	*d = CustomDate(date)
	return nil
}

func (d *CustomDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(*d).Format("2006-01-02"))
}
