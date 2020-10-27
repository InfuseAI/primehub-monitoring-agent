package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"primehub-monitoring-agent/monitoring"
	"syscall"
	"time"

	"github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
)

type Monitor struct {
	interval int
	path     string
	flush    chan struct{}
	stop     chan struct{}
	stopped  chan struct{}

	cpuCollector monitoring.CpuMemoryCollector
	gpuCollector monitoring.GpuMemoryCollector

	metrics     *monitoring.Metrics
	lifetimeMax int
}

var (
	context *daemon.Context
	monitor *Monitor
)

func NewMonitor(interval int, path string, lifetimeMax int) *Monitor {
	m := Monitor{
		interval:    interval,
		path:        path,
		lifetimeMax: lifetimeMax,
	}
	m.Init()
	return &m
}

func (m *Monitor) updateMetrics() {
	log.Debug("monitor fetch metrics")
	m.metrics.Add(m.buildRecord())
}

func (m *Monitor) buildRecord() monitoring.Record {
	cpu := m.cpuCollector.Fetch()
	gpuRecords := make([]monitoring.GPURecord, m.gpuCollector.NumDevices)
	record := monitoring.Record{
		Timestamp:      time.Now().Unix(),
		CpuUtilization: cpu.Utilization,
		MemoryUsed:     cpu.Memory,
		GPURecords:     gpuRecords,
	}

	if m.gpuCollector.Available {
		r := m.gpuCollector.Fetch()
		if r.Type == monitoring.RESULT_GPU {
			for i := 0; i < m.gpuCollector.NumDevices; i++ {
				gpuRecords[i] = monitoring.GPURecord{
					Index:          r.GPU[i].Index,
					GPUUtilization: r.GPU[i].Utilization,
					MemoryUsed:     r.GPU[i].Memory,
				}
			}
		}
	}
	return record
}

func (m *Monitor) flushToFile() {
	log.Debug("monitor flush to path", m.path)
	report := monitoring.Monitoring{
		Spec: monitoring.Spec{
			MemoryTotal: m.cpuCollector.MemoryTotal,
			GPUSpec:     m.gpuCollector.Devices,
		},
		Datasets: monitoring.Datasets{
			FifteenMinutes: m.metrics.FifteenMinutes.LastAvailable(),
			OneHour:        m.metrics.OneHour.LastAvailable(),
			ThreeHours:     m.metrics.ThreeHours.LastAvailable(),
			LifeTime:       m.metrics.LifeTime.LastAvailable(),
		},
	}

	output, _ := json.Marshal(report)
	ioutil.WriteFile("output.json", output, 0644)
}

func (m *Monitor) Init() {
	log.Debug("monitor init")
	m.flush = make(chan struct{})
	m.stop = make(chan struct{})
	m.stopped = make(chan struct{})

	m.cpuCollector = monitoring.CpuMemoryCollector{}
	m.cpuCollector.Start()
	m.gpuCollector = monitoring.GpuMemoryCollector{}
	m.gpuCollector.Start()

	m.metrics = monitoring.NewMetrics(m.lifetimeMax)
}

func (m *Monitor) Flush() {
	m.flush <- struct{}{}
}

func (m *Monitor) Worker() {
	log.Debug("monitor start")
	ticker := time.NewTicker(time.Duration(m.interval) * time.Second)
LOOP:
	// TODO this might have race-condition
	for {
		select {
		case <-ticker.C:
			m.updateMetrics()
		case <-m.flush:
			m.flushToFile()
		case <-m.stop:
			m.flushToFile()
			break LOOP
		}
	}
	m.stopped <- struct{}{}

}

func (m *Monitor) Stop() {
	log.Debug("monitor stop")
	m.stop <- struct{}{}
	<-m.stopped
	log.Debug("monitor stopped")
	m.cpuCollector.Stop()
	m.gpuCollector.Stop()
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func termHandler(sig os.Signal) error {
	log.Infof("signal by %v ...", sig)
	if monitor != nil {
		monitor.Stop()
	}
	if context != nil {
		context.Release()
		log.Info("daemon stopped")
	}
	return daemon.ErrStop
}

func flushHandler(sig os.Signal) error {
	log.Infof("manually flush by %v", sig)
	if monitor != nil {
		monitor.Flush()
	}
	return nil
}

func main() {
	var debug bool
	var isForeground bool
	var lifetimeMax int
	phJobName := getEnv("PHJOB_NAME", "job-test")
	flushPath := fmt.Sprintf("/phfs/jobArtifacts/%s/.metadata/monitoring", phJobName)

	flag.BoolVar(&debug, "debug", false, "Enable debug mod")
	flag.BoolVar(&isForeground, "D", false, "Run the agent in foreground")
	flag.StringVar(&flushPath, "path", flushPath, "Path of flush file")

	// 4 week: 5m → 4 * 7 * 24 * 60 * 60 / 300 = 8064 points
	flag.IntVar(&lifetimeMax, "lifetime-max", 8064, "Max data in the lifetime buffer")
	flag.Parse()

	context = &daemon.Context{
		PidFileName: "usage-agent.pid",
		PidFilePerm: 0644,
		LogFileName: "usage-agent.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        os.Args,
	}
	log.SetFormatter(&log.TextFormatter{})
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Demonize the usage-agent
	if isForeground == false {
		d, err := context.Reborn()
		if err != nil {
			log.Fatal("Unable to run: ", err)
		}
		if d != nil {
			// Parent process
			return
		}
		defer context.Release()
		log.Info("daemon started")
	}

	// Setup signal handler
	daemon.SetSigHandler(flushHandler, syscall.SIGHUP)
	daemon.SetSigHandler(termHandler, syscall.SIGTERM)
	daemon.SetSigHandler(termHandler, syscall.SIGQUIT)
	daemon.SetSigHandler(termHandler, syscall.SIGINT)

	log.Debugf("path: %s", flushPath)
	log.Debugf("debug: %v", debug)
	log.Debugf("isForeground: %v", isForeground)

	monitor = NewMonitor(10, flushPath, lifetimeMax)

	// Run MainLoop as worker thread
	go monitor.Worker()

	// Handle the signals
	err := daemon.ServeSignals()
	if err != nil {
		log.Errorf("Error: %s", err.Error())
	}
	return
}
