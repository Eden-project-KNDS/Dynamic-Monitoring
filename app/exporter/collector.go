package exporter

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	monitor "slurm-monitor/monitor"
)

// GPUSource is anything that ca tell us the current state of the GPU
type GPUSource interface {
	GPUs() []monitor.GPUStat
}

var gpuUtilizationDesc = prometheus.NewDesc(
	"eden_gpu_utilization_ratio",
	"GPU utilization (0-1), with the job the card is allocated to.",
	[]string{"gpu", "uuid", "job_id"},
	nil,
)

var gpuMemoryUsedDesc = prometheus.NewDesc(
	"eden_gpu_memory_used_bytes",
	"GPU memory used, with the job the card is allocated to.",
	[]string{"gpu", "uuid", "job_id"},
	nil,
)

// Collector produces eden_* metrics on every scrape
type Collector struct {
	gpus GPUSource
}

func NewCollector(gpus GPUSource) *Collector {
	return &Collector{gpus: gpus}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- gpuUtilizationDesc
	ch <- gpuMemoryUsedDesc
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	for _, g := range c.gpus.GPUs() {
		ch <- prometheus.MustNewConstMetric(
			gpuUtilizationDesc,
			prometheus.GaugeValue,
			float64(g.UtilPercent)/100,
			strconv.Itoa(g.Index), g.UUID, g.JobID,
		)

		ch <- prometheus.MustNewConstMetric(
			gpuMemoryUsedDesc,
			prometheus.GaugeValue,
			float64(g.MemoryUsed),
			strconv.Itoa(g.Index), g.UUID, g.JobID,
		)
	}
}
