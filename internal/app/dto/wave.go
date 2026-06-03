package dto

import "time"

type WaveDTO struct {
	ID               uint      `json:"id"`
	WaveNo           string    `json:"waveNo"`
	Name             string    `json:"name"`
	WaveType         string    `json:"waveType"`
	LifecycleStage   string    `json:"lifecycleStage"`
	ProgressSnapshot string    `json:"progressSnapshot"`
	Notes            string    `json:"notes"`
	LevelTags        string    `json:"levelTags"`
	CreatedAt        time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt        time.Time `json:"updatedAt" ts_type:"string"`
}

type CreateWaveInput struct {
	Name string `json:"name"`
}

type WaveDashboardRowDTO struct {
	ID                      uint      `json:"id"`
	WaveNo                  string    `json:"waveNo"`
	Name                    string    `json:"name"`
	CreatedAt               time.Time `json:"createdAt" ts_type:"string"`
	ProjectedLifecycleStage string    `json:"projectedLifecycleStage"`
}
