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
		return ResourceCollectorResult{
			Type: RESULT_NONE,
		}
	}

	results := make([]ResourceCollectorResult, g.NumDevices)

	for i := 0; i < int(g.NumDevices); i++ {
		dev, err := gonvml.DeviceHandleByIndex(uint(i))
		if err != nil {
			log.Printf("\tDeviceHandleByIndex() error: %v\n", err)
			continue
		}

		minorNumber, err := dev.MinorNumber()

		if err != nil {
			log.Printf("\tdev.MinorNumber() error: %v\n", err)
			continue
		}

		gpuUtilization, memoryUtilization, err := dev.UtilizationRates()
		if err != nil {
			log.Printf("\tdev.UtilizationRates() error: %v\n", err)
			continue
		}

		result := results[i]
		result.Index = int(minorNumber)
		result.Utilization = int(gpuUtilization)
		result.Memory = int(memoryUtilization)
	}

	return ResourceCollectorResult{
		GPU:  results,
		Type: RESULT_GPU,
	}
}
