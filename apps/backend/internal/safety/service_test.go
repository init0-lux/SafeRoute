package safety

import "time"

func serviceValueTestNow(service *Service, now time.Time) {
	service.now = func() time.Time {
		return now
	}
}
