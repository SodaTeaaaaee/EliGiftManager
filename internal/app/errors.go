package app

import "errors"

var (
	ErrWaveClosed        = errors.New("wave is closed")
	ErrAlreadyAssigned   = errors.New("fact line already assigned")
	ErrOrderExported     = errors.New("exported factory order cannot be voided")
	ErrOrderNotGenerated = errors.New("factory order is not generated")
	ErrOrderAlreadyOpen  = errors.New("an open factory order already exists for this wave and factory")
	ErrInvalidSelector   = errors.New("entitlement selector is not one of platform_level, wave_all, instance")
	ErrNothingToSubmit   = errors.New("no submittable fulfillment results")
	ErrTrackingRetired   = errors.New("tracking id is retired")
	ErrUnknownTracking   = errors.New("unknown tracking id")
)
