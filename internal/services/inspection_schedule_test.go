package services

import (
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestBuildInspectionSchedule(t *testing.T) {
	today := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
	}

	cases := []struct {
		name         string
		autoYear     int64
		approved     []int64
		rejected     []int64
		today        time.Time
		wantNextYear int64
		wantDue      string
		wantStatus   string
	}{
		{
			name:         "sin inspecciones usa año modelo + 2",
			autoYear:     2026,
			today:        today(2026, time.June, 1),
			wantNextYear: 2028,
			wantDue:      "2028-01-01",
			wantStatus:   models.InspectionStatusOK,
		},
		{
			name:         "aprobada avanza al siguiente año",
			autoYear:     2026,
			approved:     []int64{2028},
			today:        today(2028, time.February, 1),
			wantNextYear: 2029,
			wantDue:      "2029-01-01",
			wantStatus:   models.InspectionStatusOK,
		},
		{
			name:         "vencida después del 1 de enero",
			autoYear:     2026,
			today:        today(2028, time.January, 2),
			wantNextYear: 2028,
			wantDue:      "2028-01-01",
			wantStatus:   models.InspectionStatusOverdue,
		},
		{
			name:         "próxima a 60 días",
			autoYear:     2026,
			today:        today(2027, time.November, 2),
			wantNextYear: 2028,
			wantDue:      "2028-01-01",
			wantStatus:   models.InspectionStatusDueSoon,
		},
		{
			name:         "el mismo día del vencimiento es próxima",
			autoYear:     2026,
			today:        today(2028, time.January, 1),
			wantNextYear: 2028,
			wantDue:      "2028-01-01",
			wantStatus:   models.InspectionStatusDueSoon,
		},
		{
			name:         "hueco en los años aprobados",
			autoYear:     2026,
			approved:     []int64{2028, 2030},
			today:        today(2030, time.June, 1),
			wantNextYear: 2029,
			wantDue:      "2029-01-01",
			wantStatus:   models.InspectionStatusOverdue,
		},
		{
			name:         "rechazada no completa el ciclo",
			autoYear:     2026,
			approved:     []int64{2028},
			rejected:     []int64{2029},
			today:        today(2030, time.February, 1),
			wantNextYear: 2029,
			wantDue:      "2029-01-01",
			wantStatus:   models.InspectionStatusOverdue,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schedule := BuildInspectionSchedule(tc.autoYear, tc.approved, tc.rejected, tc.today)
			if schedule.NextDueYear != tc.wantNextYear {
				t.Errorf("next_due_year: esperado %d, got %d", tc.wantNextYear, schedule.NextDueYear)
			}
			if schedule.NextDueDate != tc.wantDue {
				t.Errorf("next_due_date: esperado %s, got %s", tc.wantDue, schedule.NextDueDate)
			}
			if schedule.Status != tc.wantStatus {
				t.Errorf("status: esperado %s, got %s", tc.wantStatus, schedule.Status)
			}
			if schedule.FirstDueYear != tc.autoYear+2 {
				t.Errorf("first_due_year: esperado %d, got %d", tc.autoYear+2, schedule.FirstDueYear)
			}
		})
	}
}

func TestBuildInspectionScheduleDeduplicaYOrdena(t *testing.T) {
	schedule := BuildInspectionSchedule(
		2026,
		[]int64{2029, 2028, 2029},
		[]int64{2030, 2030},
		time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
	)
	if len(schedule.ApprovedYears) != 2 || schedule.ApprovedYears[0] != 2028 || schedule.ApprovedYears[1] != 2029 {
		t.Fatalf("años aprobados inesperados: %v", schedule.ApprovedYears)
	}
	if len(schedule.RejectedYears) != 1 || schedule.RejectedYears[0] != 2030 {
		t.Fatalf("años rechazados inesperados: %v", schedule.RejectedYears)
	}
}
