package monitor

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// tu znajdują się pliki
const PATH string = "/sys/fs/cgroup/system.slice/slurmstepd.scope/"
const PATTERN string = "job_"

type jobData struct {
	user      string
	node      string
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
	mu         sync.RWMutex // the discovery loop writes the map, scrapes read them
	JobCache   map[string]jobData
	JobPid     map[string][]uint32
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

func (jobs *allJobs) getNewJobs(nodeName string) {

	entries, err := os.ReadDir(PATH)
	if err != nil {
		log.Println("Something went wrong when reading cgroups")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), PATTERN) && !keyExists(jobs.JobCache, getJobId(entry.Name())) {
			jobId := getJobId(entry.Name())
			cmd := exec.Command("scontrol", "show", "job", jobId)
			var out bytes.Buffer
			cmd.Stdout = &out

			err := cmd.Run()
			if err != nil {
				log.Printf("Error getting scrontrol data from job : %s", jobId)
				continue
			}

			meta := jobData{node: nodeName}
			output := out.String()
			rawUser := parseScontrolValue(output, "UserId")

			if idx := strings.Index(rawUser, "("); idx != -1 {
				meta.user = rawUser[:idx]
			} else {
				meta.user = rawUser
			}
			meta.account = parseScontrolValue(output, "Account")
			meta.partition = parseScontrolValue(output, "Partition")
			meta.name = parseScontrolValue(output, "JobName")
			meta.cpus, _ = strconv.Atoi(parseScontrolValue(output, "NumCPUs"))

			jobs.mu.Lock()
			jobs.JobCache[jobId] = meta
			jobs.JobPid[jobId] = []uint32{}
			jobs.JobMetrics[jobId] = cpuMetrics{}
			jobs.mu.Unlock()
			fmt.Printf("Found new job: %s USER: %s ACCOUNT: %s PARTITION: %s NAME: %s CPUS: %d\n",
				jobId, meta.user, meta.account, meta.partition, meta.name, meta.cpus)
		}
	}

}

func (jobs *allJobs) clearOldJobs() {
	jobs.mu.Lock()
	defer jobs.mu.Unlock()

	for jobId, _ := range jobs.JobCache {
		_, err := os.Stat(PATH + PATTERN + jobId)
		if err != nil { // file doesn't exist -> clearing maps
			fmt.Printf("Job %s is not longer valid\n", jobId)
			delete(jobs.JobPid, jobId)
			delete(jobs.JobMetrics, jobId)
			delete(jobs.JobCache, jobId)
		}
	}
}

func getAllPids(jobId string) []uint32 {
	var pids []uint32

	jobDir := PATH + PATTERN + jobId

	err := filepath.WalkDir(jobDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "cgroup.procs" {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				fmt.Printf("Something went wrong when getting file with pids for job %s\n", jobId)
				return nil
			}
			var currentPid uint32 = 0
			for _, b := range data {
				if b >= '0' && b <= '9' {
					currentPid = currentPid*10 + uint32(b-'0')
				} else if b == '\n' && currentPid > 0 {
					pids = append(pids, currentPid)
					currentPid = 0
				}
			}
			if currentPid > 0 {
				pids = append(pids, currentPid)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Something went wrong when getting pids for job %s\n", jobId)
		return nil
	}
	return pids
}

func (jobs *allJobs) getAllPidsFromJob() {
	jobs.mu.Lock()
	defer jobs.mu.Unlock()

	fmt.Println("Looking for PIDS")
	for jobId, _ := range jobs.JobPid {
		jobs.JobPid[jobId] = getAllPids(jobId)
		fmt.Printf("PIDS from JOB %s:%v \n", jobId, jobs.JobPid[jobId])
	}

}

// parseScontrolValue wyciąga wartość klucza w formacie Key=Value z wyjścia scontrol
func parseScontrolValue(input, key string) string {
	re := regexp.MustCompile(fmt.Sprintf(`\b%s=([^\s]+)`, regexp.QuoteMeta(key)))
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

// readUint reads a cgroup file that holds a single number, e.g. memory.current.
// A missing file (e.g. memory.peak on kernels < 5.19) gives 0.
func readUint(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// readStatValue reads one "key value" line from a cgroup file such as cpu.stat.
func readStatValue(path, key string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == key {
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			return value
		}
	}
	return 0
}

func (jobs *allJobs) getCPUData(nodeName string, flag int) []JobStat {

	var jobStats []JobStat = make([]JobStat, 0)

	fmt.Printf("Size of JobCache : %d\n", len(jobs.JobCache))

	jobs.getNewJobs(nodeName)
	jobs.clearOldJobs()
	jobs.getAllPidsFromJob()

	if flag == 1 {
		return jobStats
	}
	return nil
}

func (jobs *allJobs) Jobs() []JobStat {
	return jobs.getCPUData("gpunode3", 1)
}

// key is job_id number to search for a dir need to add PATTERN
func newAllJobs() *allJobs {
	return &allJobs{
		JobCache:   make(map[string]jobData),
		JobPid:     make(map[string][]uint32),
		JobMetrics: make(map[string]cpuMetrics),
	}
}

// run is the discovery loop: it finds new jobs, removes finished ones and refreshes PIDs.
func (jobs *allJobs) run(nodeName string) {
	fmt.Println("Init CPU Monitoring")

	ticker := time.NewTicker(time.Duration(TimeInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		//obsługa błedów

		case t := <-ticker.C:
			{
				fmt.Printf("CPU RAM %s\n", t.Format("2006-01-02 15:04:05"))
				jobs.getCPUData(nodeName, 0)
			}
		}
	}
}
