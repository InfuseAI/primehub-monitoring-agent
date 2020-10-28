package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"primehub-monitoring-agent/monitoring"
	"syscall"
	"time"

	"github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
)

type Monitor struct {
	updateInterval int
	flushPeriod    int
	path           string
	flush          chan struct{}
	stop           chan struct{}
	stopped        chan struct{}

	cpuCollector monitoring.CpuMemoryCollector
	gpuCollector monitoring.GpuMemoryCollector

	metrics     *monitoring.Metrics
	lifetimeMax int
}

var (
	context *daemon.Context
	monitor *Monitor
)

func NewMonitor(updateInterval int, path string, lifetimeMax int, flushPeriod int) *Monitor {
	m := Monitor{
		updateInterval: updateInterval,
		path:           path,
		lifetimeMax:    lifetimeMax,
		flushPeriod:    flushPeriod,
	}
	m.Init()
	return &m
}

func (m *Monitor) updateMetrics() {
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
		for i := 0; i < m.gpuCollector.NumDevices; i++ {
			gpuRecords[i] = monitoring.GPURecord{
				Index:          r.GPU[i].Index,
				GPUUtilization: r.GPU[i].Utilization,
				MemoryUsed:     r.GPU[i].Memory,
			}
		}
	}

	if log.GetLevel() == log.DebugLevel {
		log.Debugf("[BuildRecord] CPU: %d, MemoryUsed: %d", record.CpuUtilization, record.MemoryUsed)
		for i := 0; i < len(record.GPURecords); i++ {
			log.Debugf("[BuildRecord] GPU[%d], GPUUtilization: %d, MemoryUsed: %d",
				record.GPURecords[i].Index, record.GPURecords[i].GPUUtilization, record.GPURecords[i].MemoryUsed)
		}
	}
	return record
}

func (m *Monitor) flushToFile() {
	log.Debugf("[FlushRecord] Path: %s", m.path)
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
	ioutil.WriteFile(m.path, output, 0644)
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
	counter := 0
	ticker := time.NewTicker(time.Duration(m.updateInterval) * time.Second)
LOOP:
	// TODO this might have race-condition
	for {
		select {
		case <-ticker.C:
			m.updateMetrics()
			counter++
			if counter == m.flushPeriod {
				m.flushToFile()
				counter = 0
			}
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
	var updateInterval int
	var flushPeriod int
	phJobName := getEnv("PHJOB_NAME", "job-test")
	flushPath := fmt.Sprintf("/phfs/jobArtifacts/%s/.metadata/monitoring", phJobName)

	flag.BoolVar(&debug, "debug", false, "Enable debug mod")
	flag.BoolVar(&isForeground, "D", false, "Run the agent in foreground")
	flag.StringVar(&flushPath, "path", flushPath, "Path of flush file")
	flag.IntVar(&updateInterval, "updateInterval", 10, "Interval seconds of update metrics")
	flag.IntVar(&flushPeriod, "flushPeriod", 3, "Period of interval for flushing metrics to file")

	// 4 week: 5m → 4 * 7 * 24 * 60 * 60 / 300 = 8064 points
	flag.IntVar(&lifetimeMax, "lifetime-max", 8064, "Max data in the lifetime buffer")
	flag.Parse()

	context = &daemon.Context{
		PidFileName: ".monitoring-agent.pid",
		PidFilePerm: 0644,
		LogFileName: ".monitoring-agent.log",
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

	// Check flush path exist or not
	if _, err := os.Stat(filepath.Dir(flushPath)); os.IsNotExist(err) {
		log.Warnf("Directory %s doesn't exist, fallback to 'monitoring.json'", filepath.Dir(flushPath))
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		flushPath = filepath.Join(pwd, "monitoring.json")
	}

	// Setup signal handler
	daemon.SetSigHandler(flushHandler, syscall.SIGHUP)
	daemon.SetSigHandler(termHandler, syscall.SIGTERM)
	daemon.SetSigHandler(termHandler, syscall.SIGQUIT)
	daemon.SetSigHandler(termHandler, syscall.SIGINT)

	log.Debug(monitoring.GetVersion())
	log.Debugf("path: %s", flushPath)
	log.Debugf("debug: %v", debug)
	log.Debugf("isForeground: %v", isForeground)

	monitor = NewMonitor(updateInterval, flushPath, lifetimeMax, flushPeriod)

	// Run MainLoop as worker thread
	go monitor.Worker()

	// Handle the signals
	err := daemon.ServeSignals()
	if err != nil {
		log.Errorf("Error: %s", err.Error())
	}
	return
}
