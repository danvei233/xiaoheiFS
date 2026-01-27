package usecase

import "xiaoheiplay/internal/domain"

func MapAutomationState(state int) domain.VPSStatus {
	switch state {
	case 0, 1, 13:
		return domain.VPSStatusProvisioning
	case 2:
		return domain.VPSStatusRunning
	case 3:
		return domain.VPSStatusStopped
	case 4:
		return domain.VPSStatusReinstalling
	case 5:
		return domain.VPSStatusReinstallFailed
	case 10:
		return domain.VPSStatusLocked
	default:
		return domain.VPSStatusUnknown
	}
}
