package monitoring

import (
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

const UnlimitedMemory = 9223372036854771712

type CpuMemoryCollector struct {
	// The path should be one of
	// /sys/fs/cgroup/cpuacct/cpuacct.usage
	// /sys/fs/cgroup/CPU/cpuacct.usage
	CpuAcctUsagePath string
	UpdateTime       time.Time
	StartedTime      time.Time
	CpuAcctValue     int64
	CpuUsageValue    int
	MemoryUsage      int
	MemoryTotal      int
	StopFlag         chan int
}

func (r *CpuMemoryCollector) Fetch() ResourceCollectorResult {
	return ResourceCollectorResult{
		Utilization: r.CpuUsageValue,
		Memory:      r.MemoryUsage,
		Index:       0,
		GPU:         nil,
	}
}

func (r *CpuMemoryCollector) Start() {
	r.StartedTime = time.Now()
	r.StopFlag = make(chan int)

	if _, err := os.Stat("/tmp/dev-cpuacct.usage"); err == nil {
		r.CpuAcctUsagePath = "/tmp/dev-cpuacct.usage"
	} else if _, err := os.Stat("/sys/fs/cgroup/cpuacct/cpuacct.usage"); err == nil {
		r.CpuAcctUsagePath = "/sys/fs/cgroup/cpuacct/cpuacct.usage"
	} else if _, err := os.Stat("/sys/fs/cgroup/CPU/cpuacct.usage"); err == nil {
		r.CpuAcctUsagePath = "/sys/fs/cgroup/CPU/cpuacct.usage"
	}

	memoryTotal, err := ReadNumber("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	if err != nil {
		log.Errorf("Cannot get memory total from %s", "/sys/fs/cgroup/memory/memory.limit_in_bytes")
	}
	if memoryTotal == UnlimitedMemory {
		log.Warnf("Found unlimited memory settings (%d), keep MemoryTotal as 0", UnlimitedMemory)
	} else {
		r.MemoryTotal = int(memoryTotal / 1024 * 1024)
		log.Infof("Set MemoryTotal %d MB", r.MemoryTotal)
	}

	go r.update()
}

func (r *CpuMemoryCollector) update() {
	ticker := time.NewTicker(time.Duration(5) * time.Second)
	for {
		select {
		case <-ticker.C:
			r.updateCpuUsage()
			r.updateMemoryUsage()
		case <-r.StopFlag:
			break
		}
	}

}

func (r *CpuMemoryCollector) updateCpuUsage() {
	number, err := ReadNumber(r.CpuAcctUsagePath)
	if err == nil {
		if r.UpdateTime.IsZero() {
			r.updateCpuCurrentValue(number)
		} else {
			// calculate usage before update current values
			// CPU Usage = Δ cpuacct.usage / duration
			duration := int(time.Now().Sub(r.UpdateTime).Nanoseconds())
			r.CpuUsageValue = int(number-r.CpuAcctValue) * 100 / duration
			r.updateCpuCurrentValue(number)
		}
	}
}

func (r *CpuMemoryCollector) updateCpuCurrentValue(number int64) {
	r.UpdateTime = time.Now()
	r.CpuAcctValue = number
}

func (r *CpuMemoryCollector) updateMemoryUsage() {
	usageInBytes, usageErr := ReadNumber("/sys/fs/cgroup/memory/memory.usage_in_bytes")
	inactive, inactiveErr := ReadTotalInactiveFile("/sys/fs/cgroup/memory/memory.stat")
	if usageErr == nil && inactiveErr == nil {
		r.MemoryUsage = int((usageInBytes - inactive) / (1024 * 1024))
	}
}

func (r *CpuMemoryCollector) Stop() {
	r.StopFlag <- 1
}
