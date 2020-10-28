package monitoring

type GPUSpec struct {
	Index       int   `json:"index"`
	MemoryTotal int64 `json:"mem_total"`
}

type Spec struct {
	MemoryTotal int64     `json:"mem_total"`
	GPUSpec     []GPUSpec `json:"GPU"`
}

type GPURecord struct {
	Index          int   `json:"index"`
	MemoryUsed     int64 `json:"mem_used"`
	GPUUtilization int   `json:"gpu_util"`
}

type Record struct {
	Timestamp      int64       `json:"timestamp"`
	CpuUtilization int         `json:"cpu_util"`
	MemoryUsed     int64       `json:"mem_used"`
	GPURecords     []GPURecord `json:"GPU"`
}

type Datasets struct {
	FifteenMinutes []Record `json:"15m"`
	OneHour        []Record `json:"1h"`
	ThreeHours     []Record `json:"3h"`
	LifeTime       []Record `json:"lifetime"`
}

type Monitoring struct {
	Spec     Spec     `json:"spec"`
	Datasets Datasets `json:"datasets"`
}
