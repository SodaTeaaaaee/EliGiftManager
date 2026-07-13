package domain

import "context"

// WaveLifecycleRepository defines write-path persistence operations for wave
// lifecycle management (rename/notes editing and explicit closure) that are not
// covered by WaveRepository, whose only lifecycle write is the projection-driven
// UpdateLifecycle (stage + progress_snapshot together).
//
// Implemented by the same concrete type as WaveRepository — see
// infra.NewWaveLifecycleRepository (internal/infra/wave_lifecycle_repo.go).
type WaveLifecycleRepository interface {
	// UpdateWaveFields persists the operator-editable metadata fields
	// (name/notes/levelTags) for a wave. Used by UpdateWave and, for closure
	// notes, by CloseWave.
	UpdateWaveFields(ctx context.Context, waveID uint, name, notes, levelTags string) error

	// TransitionLifecycleStage sets the wave's lifecycle_stage column directly,
	// leaving progress_snapshot untouched (unlike WaveRepository.UpdateLifecycle,
	// which always overwrites both). Used by CloseWave to move a wave into the
	// closed stage without clobbering its last computed progress snapshot.
	TransitionLifecycleStage(ctx context.Context, waveID uint, stage string) error
}
