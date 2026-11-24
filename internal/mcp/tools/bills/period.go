package bills

import (
	"fmt"
	"strings"
	"time"
)

type period struct {
	Year  int
	Month int
}

func parsePeriod(raw string) (period, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := time.Parse("2006-01", raw)
	if err != nil || len(raw) != 7 {
		return period{}, fmt.Errorf("el período debe tener formato YYYY-MM")
	}
	return period{Year: parsed.Year(), Month: int(parsed.Month())}, nil
}

func (p period) String() string {
	return fmt.Sprintf("%04d-%02d", p.Year, p.Month)
}

func (p period) before(other period) bool {
	if p.Year != other.Year {
		return p.Year < other.Year
	}
	return p.Month < other.Month
}

func (p period) after(other period) bool {
	if p.Year != other.Year {
		return p.Year > other.Year
	}
	return p.Month > other.Month
}

func currentPeriod(now time.Time) period {
	return period{Year: now.Year(), Month: int(now.Month())}
}

func monthsBetween(from, to period) int {
	return (to.Year-from.Year)*12 + (to.Month - from.Month) + 1
}

// billPeriod devuelve el período calendario de una factura. Las facturas
// anuales se guardan con month=0 y su período es enero.
func billPeriodOf(year, month int) period {
	if month == 0 {
		month = 1
	}
	return period{Year: year, Month: month}
}
