package monitor

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// tu znajdują się pliki
const (
	PATH    string = "/sys/fs/cgroup/system.slice/slurmstepd.scope/"
	PATTERN string = "job_"
)

type jobData struct {
	user      string
	account   string
	partition string
	name      string
	cpus      int
}

type cpuMetrics struct {
	// RAM
	memoryCurrent uint64
	memoryAnon    uint64
	memoryPeak    uint64
	memoryLimit   uint64
	oomKills      uint64

	// CPU
	cpuUsageUsec  uint64
	cpuUserUsec   uint64
	cpuSystemUsec uint64
	throttledUsec uint64
}

type allJobs struct {
	nodeName   string
	JobCache   map[string]jobData
	JobMetrics map[string]cpuMetrics
}

func keyExists(m map[string]jobData, key string) bool {
	_, ok := m[key]
	return ok
}

// this function assumes that folderName starts with job_ prefix
func getJobId(folderName string) string {
	return folderName[4:]
}

func (jobs *allJobs) getNewJobs() {
	entries, err := os.ReadDir(PATH)
	if err != nil {
		log.Printf("Couldn't read cgroups at %s: %v", PATH, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), PATTERN) && !keyExists(jobs.JobCache, getJobId(entry.Name())) {
			jobId := getJobId(entry.Name())
			cmd := exec.Command("/usr/local/slurm/latest/bin/scontrol", "show", "job", jobId)
			var out bytes.Buffer
			cmd.Stdout = &out
			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err != nil {
				log.Printf("Couldn't get scontrol data for job %s: %v; stderr: %s", jobId, err, strings.TrimSpace(stderr.String()))
				continue
			}

			meta := jobData{}
			output := out.String()
			rawUser, err := tryParseScontrolValue(output, "UserId")
			if err != nil {
				log.Printf("Couldn't parse scontrol UserId for job %s: %v", jobId, err)
				continue
			}

			if idx := strings.Index(rawUser, "("); idx != -1 {
				meta.user = rawUser[:idx]
			} else {
				meta.user = rawUser
			}
			meta.account, err = tryParseScontrolValue(output, "Account")
			if err != nil {
				log.Printf("Couldn't parse scontrol Account for job %s: %v", jobId, err)
				continue
			}
			meta.partition, err = tryParseScontrolValue(output, "Partition")
			if err != nil {
				log.Printf("Couldn't parse scontrol Partition for job %s: %v", jobId, err)
				continue
			}
			meta.name, err = tryParseScontrolValue(output, "JobName")
			if err != nil {
				log.Printf("Couldn't parse scontrol JobName for job %s: %v", jobId, err)
				continue
			}
			cpus, err := tryParseScontrolValue(output, "NumCPUs")
			if err != nil {
				log.Printf("Couldn't parse scontrol NumCPUs for job %s: %v", jobId, err)
				continue
			}
			meta.cpus, err = strconv.Atoi(cpus)
			if err != nil {
				log.Printf("Couldn't parse scontrol NumCPUs %q for job %s: %v", cpus, jobId, err)
				continue
			}

			jobs.JobCache[jobId] = meta
			jobs.JobMetrics[jobId] = cpuMetrics{}

			fmt.Printf("Found new job: %s USER: %s ACCOUNT: %s PARTITION: %s NAME: %s CPUS: %d\n",
				jobId, meta.user, meta.account, meta.partition, meta.name, meta.cpus)
		}
	}
}

func (jobs *allJobs) clearOldJobs() {
	for jobId := range jobs.JobCache {
		_, err := os.Stat(PATH + PATTERN + jobId)
		if err != nil { // file doesn't exist -> clearing maps
			fmt.Printf("Job %s is not longer valid\n", jobId)
			delete(jobs.JobMetrics, jobId)
			delete(jobs.JobCache, jobId)
		}
	}
}

// parseScontrolValue wyciąga wartość klucza w formacie Key=Value z wyjścia scontrol
func tryParseScontrolValue(input, key string) (string, error) {
	re := regexp.MustCompile(fmt.Sprintf(`\b%s=([^\s]+)`, regexp.QuoteMeta(key)))
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1], nil
	}
	return "", fmt.Errorf("Couldn't find key `%s`", key)
}

// tryReadCGroupsAsUint reads a cgroup file that holds a single number, e.g. memory.current.
// A missing file (e.g. memory.peak on kernels < 5.19) returns an error.
func tryReadCGroupsAsUint(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("Couldn't read the file at the path `%s`: %w", path, err)
	}
	raw := strings.TrimSpace(string(data))
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Couldn't parse the input %q at `%s` as a uint: %w", raw, path, err)
	}
	return value, nil
}

// tryReadStatValue reads one "key value" line from a cgroup file such as cpu.stat.
func tryReadStatValue(path, key string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("Couldn't read file at `%s`: %w", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == key {
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("Couldn't parse value %q for key `%s` at `%s`: %w", fields[1], key, path, err)
			}
			return value, nil
		}
	}
	return 0, fmt.Errorf("Didn't find key `%s` at path `%s`", key, path)
}

func readJobStat(jobId string, meta jobData, dir string) JobStat {
	dirMemoryCurrent := dir + "memory.current"
	memoryCurrent, memoryCurrentErr := tryReadCGroupsAsUint(dirMemoryCurrent)
	if memoryCurrentErr != nil {
		log.Printf("Couldn't read memory.current for job %s: %v", jobId, memoryCurrentErr)
	}

	dirMemoryPeak := dir + "memory.peak"
	memoryPeak, memoryPeakErr := tryReadCGroupsAsUint(dirMemoryPeak)
	if memoryPeakErr != nil {
		log.Printf("Couldn't read memory.peak for job %s: %v", jobId, memoryPeakErr)
	}

	dirMemoryMax := dir + "memory.max"
	memoryMax, memoryMaxErr := tryReadCGroupsAsUint(dirMemoryMax)
	if memoryMaxErr != nil {
		log.Printf("Couldn't read memory.max for job %s: %v", jobId, memoryMaxErr)
	}

	dirCPUStat := dir + "cpu.stat"
	cpuUsage, cpuUsageErr := tryReadStatValue(dirCPUStat, "usage_usec")
	if cpuUsageErr != nil {
		log.Printf("Couldn't read cpu.stat usage_usec for job %s: %v", jobId, cpuUsageErr)
	}

	return JobStat{
		JobID:         jobId,
		User:          meta.user,
		Account:       meta.account,
		Partition:     meta.partition,
		Name:          meta.name,
		CPUsAllocated: meta.cpus,
		CPUUsageUsec:  cpuUsage,
		MemoryCurrent: memoryCurrent,
		MemoryPeak:    memoryPeak,
		MemoryLimit:   memoryMax,

		CPUUsageValid:      cpuUsageErr == nil,
		MemoryCurrentValid: memoryCurrentErr == nil,
		MemoryPeakValid:    memoryPeakErr == nil,
		MemoryLimitValid:   memoryMaxErr == nil,
	}
}

// Jobs returns the jobs found by the discovery loop, with usage read from cgroup right now (on every scrape).
func (jobs *allJobs) Jobs() []JobStat {
	jobs.getNewJobs()
	jobs.clearOldJobs()
	fmt.Printf("Size of JobCache : %d\n", len(jobs.JobCache))

	stats := make([]JobStat, 0, len(jobs.JobCache))
	for jobId, meta := range jobs.JobCache {
		dir := PATH + PATTERN + jobId + "/"
		if _, err := os.Stat(dir); err != nil {
			continue // the job has just ended, the loop will remove it soon
		}
		stats = append(stats, readJobStat(jobId, meta, dir))
	}
	return stats
}

// key is job_id number to search for a dir need to add PATTERN
func NewAllJobs() *allJobs {
	return &allJobs{
		JobCache:   make(map[string]jobData),
		JobMetrics: make(map[string]cpuMetrics),
	}
}
