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
	ErrLineFrozen        = errors.New("fact line entered a factory order and can no longer move")
	ErrRevisionPending   = errors.New("fact revises another fact and is pending; apply or dismiss the revision first")
	ErrNotRevision       = errors.New("fact is not a pending revision")
	ErrRevisionApplied   = errors.New("revision was already applied")
)
