package pricing

// NormalizeReservedRate calculates the effective hourly rate for Reserved Instances
// by combining upfront and hourly charges
func NormalizeReservedRate(upfront, hourly float64, termYears int) float64 {
	termHours := float64(termYears * 365 * 24)
	effectiveHourly := (upfront / termHours) + hourly
	return effectiveHourly
}

// GetTermYears extracts the number of years from a term key (e.g., "1yr_AU" -> 1)
func GetTermYears(termKey string) int {
	if len(termKey) >= 3 && termKey[:3] == "1yr" {
		return 1
	}
	if len(termKey) >= 3 && termKey[:3] == "3yr" {
		return 3
	}
	return 0
}
