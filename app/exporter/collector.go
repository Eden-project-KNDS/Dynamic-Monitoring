package exporter

import (
	helper "slurm-monitor/helper"
	monitor "slurm-monitor/monitor"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// GPUSource is anything that can tell us the current state of the GPU.
type GPUSource interface {
	GPUs() []monitor.GPUStat
}

// JobSource is anything that can tell us which jobs run on this node and what they use.
type JobSource interface {
	Jobs() []monitor.JobStat
}

var (
	gpuLabels = []string{"gpu", "uuid", "job_id"}
	jobLabels = []string{"job_id"}

	gpuUtilizationDesc = prometheus.NewDesc(
		"eden_gpu_utilization_ratio",
		"GPU utilization (0-1), with the job the card is allocated to.",
		gpuLabels, nil,
	)
	gpuMemoryUsedDesc = prometheus.NewDesc(
		"eden_gpu_memory_used_bytes",
		"GPU memory used, with the job the card is allocated to.",
		gpuLabels, nil,
	)

	jobInfoDesc = prometheus.NewDesc(
		"eden_job_info",
		"Job description from Slurm (value is always 1).",
		[]string{"job_id", "user", "account", "partition", "job_name"}, nil,
	)
	jobCPUUsageDesc = prometheus.NewDesc(
		"eden_job_cpu_usage_seconds_total",
		"CPU time used by the job.",
		jobLabels, nil,
	)
	jobCPUsAllocatedDesc = prometheus.NewDesc(
		"eden_job_cpus_allocated",
		"CPU cores allocated to the job.",
		jobLabels, nil,
	)
	jobMemoryUsedDesc = prometheus.NewDesc(
		"eden_job_memory_used_bytes",
		"Memory currently used by the job.",
		jobLabels, nil,
	)
	jobMemoryPeakDesc = prometheus.NewDesc(
		"eden_job_memory_peak_bytes",
		"Peak memory used by the job.",
		jobLabels, nil,
	)
	jobMemoryAllocatedDesc = prometheus.NewDesc(
		"eden_job_memory_allocated_bytes",
		"Memory allocated to the job.",
		jobLabels, nil,
	)
)

// Collector produces eden_* metrics on every scrape.
type Collector struct {
	gpus GPUSource
	jobs JobSource
}

func NewCollector(gpus GPUSource, jobs JobSource) *Collector {
	return &Collector{gpus: gpus, jobs: jobs}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- gpuUtilizationDesc
	ch <- gpuMemoryUsedDesc
	ch <- jobInfoDesc
	ch <- jobCPUUsageDesc
	ch <- jobCPUsAllocatedDesc
	ch <- jobMemoryUsedDesc
	ch <- jobMemoryPeakDesc
	ch <- jobMemoryAllocatedDesc
}

// Collect runs on every scrape and logs one summary line per scrape
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	gpuCount := c.collectGPUs(ch)
	jobCount := c.collectJobs(ch)
	helper.GeneralLogger.Printf("Scrape served: %d GPUs, %d jobs in %s",
		gpuCount, jobCount, time.Since(start).Round(time.Microsecond))
}

func (c *Collector) collectGPUs(ch chan<- prometheus.Metric) int {
	gpus := c.gpus.GPUs()
	for _, g := range gpus {
		if g.UtilPercentValid {
			ch <- prometheus.MustNewConstMetric(
				gpuUtilizationDesc,
				prometheus.GaugeValue,
				float64(g.UtilPercent)/100,
				strconv.Itoa(g.Index), g.UUID, g.JobID,
			)
		}

		if g.MemoryUsedValid {
			ch <- prometheus.MustNewConstMetric(
				gpuMemoryUsedDesc,
				prometheus.GaugeValue,
				float64(g.MemoryUsed),
				strconv.Itoa(g.Index), g.UUID, g.JobID,
			)
		}
	}
	return len(gpus)
}

func (c *Collector) collectJobs(ch chan<- prometheus.Metric) int {
	jobs := c.jobs.Jobs()
	for _, j := range jobs {
		ch <- prometheus.MustNewConstMetric(
			jobInfoDesc,
			prometheus.GaugeValue,
			1,
			j.JobID, j.User, j.Account, j.Partition, j.Name,
		)

		if j.CPUUsageValid {
			ch <- prometheus.MustNewConstMetric(
				jobCPUUsageDesc,
				prometheus.CounterValue,
				float64(j.CPUUsageUsec)/1e6, // microseconds -> seconds
				j.JobID,
			)
		}

		ch <- prometheus.MustNewConstMetric(
			jobCPUsAllocatedDesc,
			prometheus.GaugeValue,
			float64(j.CPUsAllocated),
			j.JobID,
		)

		if j.MemoryCurrentValid {
			ch <- prometheus.MustNewConstMetric(
				jobMemoryUsedDesc,
				prometheus.GaugeValue,
				float64(j.MemoryCurrent),
				j.JobID,
			)
		}

		// If memory.peak is unavailable, Grafana can use max_over_time(memory_used) instead.
		if j.MemoryPeakValid {
			ch <- prometheus.MustNewConstMetric(
				jobMemoryPeakDesc,
				prometheus.GaugeValue,
				float64(j.MemoryPeak),
				j.JobID,
			)
		}

		// No limit means nothing was reserved, so there is nothing to report.
		if j.MemoryLimitValid && j.MemoryLimit > 0 {
			ch <- prometheus.MustNewConstMetric(
				jobMemoryAllocatedDesc,
				prometheus.GaugeValue,
				float64(j.MemoryLimit),
				j.JobID,
			)
		}
	}
	return len(jobs)
}
