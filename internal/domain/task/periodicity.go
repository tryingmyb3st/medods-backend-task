package task

import (
	"errors"
	"time"
)

type PeriodType string

const (
	Daily        PeriodType = "daily"
	Monthly      PeriodType = "monthly"
	Custom       PeriodType = "custom"
	EvenOddDates PeriodType = "even_odd"
)

type Periodicity struct {
	TaskId        int64       `json:"-"`
	Type          PeriodType  `json:"period_type"`
	DaysInterval  *int64      `json:"days_interval"`
	DayOfMonth    *int64      `json:"day_of_month"`
	CustomDates   []time.Time `json:"custom_date"`
	EvenOdd       *string     `json:"even_odd"`
	LastCreatedAt *time.Time  `json:"-"`
}

func (p PeriodType) Valid() bool {
	switch p {
	case Daily, Monthly, Custom, EvenOddDates:
		return true
	default:
		return false
	}
}

func (p *Periodicity) Validate() error {
	if p.Type == Daily && p.DaysInterval == nil {
		return errors.New("days is required")
	}

	if p.DaysInterval != nil && *p.DaysInterval <= 0 {
		return errors.New("days must be positive")
	}

	if p.Type == Monthly && p.DayOfMonth == nil {
		return errors.New("day of the month is required")
	}

	if p.Type == Monthly && (*p.DayOfMonth < 1 || *p.DayOfMonth > 30) {
		return errors.New("day of the month must be greater than 0 and less than 31")
	}

	if p.Type == Custom && len(p.CustomDates) == 0 {
		return errors.New("date is required")
	}

	if p.Type == Custom && p.CustomDates != nil {
		for _, date := range p.CustomDates {
			if date.Before(time.Now()) {
				return errors.New("dates must be in the future")
			}
		}
	}

	if p.Type == EvenOddDates && p.EvenOdd == nil {
		return errors.New("even or odd choice is required")
	}

	if p.Type == EvenOddDates && p.EvenOdd != nil && *p.EvenOdd != "even" && *p.EvenOdd != "odd" {
		return errors.New("only even or odd choice is allowed")
	}

	return nil
}
