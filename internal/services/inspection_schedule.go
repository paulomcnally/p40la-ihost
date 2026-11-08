package services

import (
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// BuildInspectionSchedule calcula el cronograma de inspecciones mecánicas
// (SPEC-124). La primera inspección vence el 1 de enero del año modelo + 2 y
// luego todos los años. La próxima inspección es el primer año del calendario
// sin una inspección aprobada; las rechazadas no completan el ciclo.
func BuildInspectionSchedule(autoYear int64, approvedYears, rejectedYears []int64, today time.Time) models.InspectionSchedule {
	firstDueYear := autoYear + 2
	if firstDueYear < 1902 {
		firstDueYear = 1902
	}

	approvedSet := make(map[int64]struct{}, len(approvedYears))
	for _, year := range approvedYears {
		approvedSet[year] = struct{}{}
	}

	nextDueYear := firstDueYear
	for {
		if _, ok := approvedSet[nextDueYear]; !ok {
			break
		}
		nextDueYear++
	}

	dueDate := time.Date(int(nextDueYear), time.January, 1, 0, 0, 0, 0, time.UTC)
	todayDate := dateOnly(today)
	days := int64(dueDate.Sub(todayDate).Hours() / 24)

	status := models.InspectionStatusOK
	switch {
	case days < 0:
		status = models.InspectionStatusOverdue
	case days <= models.InspectionDueSoonDays:
		status = models.InspectionStatusDueSoon
	}

	return models.InspectionSchedule{
		FirstDueYear:  firstDueYear,
		NextDueYear:   nextDueYear,
		NextDueDate:   dueDate.Format("2006-01-02"),
		Status:        status,
		DaysUntilDue:  days,
		ApprovedYears: uniqueSortedYears(approvedYears),
		RejectedYears: uniqueSortedYears(rejectedYears),
	}
}

func uniqueSortedYears(years []int64) []int64 {
	if len(years) == 0 {
		return []int64{}
	}
	seen := make(map[int64]struct{}, len(years))
	result := make([]int64, 0, len(years))
	for _, year := range years {
		if _, ok := seen[year]; ok {
			continue
		}
		seen[year] = struct{}{}
		result = append(result, year)
	}
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j] < result[j-1]; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result
}
