package monitoring

const (
	RESULT_CPU  = "CPU"
	RESULT_GPU  = "GPU"
	RESULT_NONE = "none"
)

type ResourceCollectorResult struct {
	Utilization int
	Memory      int
	Index       int
	GPU         []ResourceCollectorResult
	Type        string
}

type ResourceCollector interface {
	Start()
	Stop()

	// return Utilization and MemoryUsage
	Fetch() ResourceCollectorResult
}
