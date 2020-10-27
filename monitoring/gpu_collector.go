package monitoring

import (
	"github.com/mindprince/gonvml"
	log "github.com/sirupsen/logrus"
)

type GpuMemoryCollector struct {
	Available  bool
	NumDevices int
	Devices    []GPUSpec
}

func (g *GpuMemoryCollector) Start() {
	err := gonvml.Initialize()
	g.Available = false
	g.Devices = make([]GPUSpec, 0)

	if err != nil {
		log.Warn(err)
		return
	}

	numDevices, err := gonvml.DeviceCount()
	log.Infof("Get %d gpu-devices", numDevices)
	if err != nil {
		log.Printf("DeviceCount() error: %v\n", err)
		defer gonvml.Shutdown()
		return
	}

	if numDevices == 0 {
		return
	}

	g.NumDevices = int(numDevices)
	g.Devices = make([]GPUSpec, numDevices)
	for i := 0; i < int(g.NumDevices); i++ {
		dev, err := gonvml.DeviceHandleByIndex(uint(i))
		if err != nil {
			log.Warn(err)
		}
		deviceIndex, _ := dev.MinorNumber()
		total, _, _ := dev.MemoryInfo()

		g.Devices[i].Index = int(deviceIndex)
		g.Devices[i].MemoryTotal = int(total / (1024 * 1024))
		log.Infof("Set device[%d] Index=%d, Memory=%d\n", i, deviceIndex, g.Devices[i].MemoryTotal)
	}
	g.Available = true
}

func (g *GpuMemoryCollector) Stop() {
	if g.Available {
		gonvml.Shutdown()
	}
}

func (g *GpuMemoryCollector) Fetch() ResourceCollectorResult {
	if !g.Available {
		return ResourceCollectorResult{}
	}

	results := make([]ResourceCollectorResult, g.NumDevices)

	for i := 0; i < int(g.NumDevices); i++ {
		dev, err := gonvml.DeviceHandleByIndex(uint(i))
		if err != nil {
			log.Debugf("DeviceHandleByIndex() error: %v", err)
			continue
		}

		minorNumber, err := dev.MinorNumber()

		if err != nil {
			log.Debugf("dev.MinorNumber() error: %v", err)
			continue
		}

		gpuUtilization, memoryUtilization, err := dev.UtilizationRates()
		if err != nil {
			log.Debugf("dev.UtilizationRates() error: %v", err)
			continue
		}

		results[i].Index = int(minorNumber)
		results[i].Utilization = int(gpuUtilization)
		results[i].Memory = int(memoryUtilization)

		log.Debugf("GPU::device [%d], Utilization: %d, Memory: %d",
			minorNumber, gpuUtilization, memoryUtilization)
	}

	return ResourceCollectorResult{
		GPU: results,
	}
}
