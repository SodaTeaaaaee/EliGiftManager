package dto

type WaveDTO struct {
	ID               uint   `json:"id"`
	WaveNo           string `json:"waveNo"`
	Name             string `json:"name"`
	WaveType         string `json:"waveType"`
	LifecycleStage   string `json:"lifecycleStage"`
	ProgressSnapshot string `json:"progressSnapshot"`
	Notes            string `json:"notes"`
	LevelTags        string `json:"levelTags"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type CreateWaveInput struct {
	Name string `json:"name"`
}

type WaveDashboardRowDTO struct {
	ID                     uint   `json:"id"`
	WaveNo                 string `json:"waveNo"`
	Name                   string `json:"name"`
	CreatedAt              string `json:"createdAt"`
	ProjectedLifecycleStage string `json:"projectedLifecycleStage"`
}
