package monitoring

type ResourceCollectorResult struct {
	Utilization int
	Memory      int64

	// used by gpu result
	Index int
	GPU   []ResourceCollectorResult
}

type ResourceCollector interface {
	Start()
	Stop()

	// return Utilization and MemoryUsage
	Fetch() ResourceCollectorResult
}
