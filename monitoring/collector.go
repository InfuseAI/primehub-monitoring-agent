package monitoring

type ResourceCollectorResult struct {
	Utilization int
	Memory      int
	Index       int
	GPU         []ResourceCollectorResult
}

type ResourceCollector interface {
	Start()
	Stop()

	// return Utilization and MemoryUsage
	Fetch() ResourceCollectorResult
}
