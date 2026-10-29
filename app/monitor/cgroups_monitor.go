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
	"strings"
	"time"
)

// tu znajdują się pliki
const PATH string = "/sys/fs/cgroup/system.slice/slurmstepd.scope/"
const PATTERN string = "job_"

type jobData struct {
	user string
	node string
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

func keyExists(m map[string]jobData, key string) bool {
	_, ok := m[key]
	return ok
}

// this function assumes that folderName starts with job_ prefix
func getJobId(folderName string) string {
	return folderName[4:]
}

func getNewJobs(JobCache map[string]jobData, nodeName string) {

	entries, err := os.ReadDir(PATH)
	if err != nil {
		log.Println("Something went wrong when reading cgroups")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), PATTERN) && !keyExists(JobCache, getJobId(entry.Name())) {
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

			JobCache[jobId] = meta

			fmt.Printf("Found new job: %s USER: %s\n", jobId, meta.user)
		}
	}

}

func clearOldJobs(JobCache map[string]jobData, JobPID map[string][]uint32, JobMetrics map[string]cpuMetrics) {

	for jobId, _ := range JobCache {
		_, err := os.Stat(PATH + PATTERN + jobId)
		if err != nil { // file doesn't exist -> clearing maps
			fmt.Printf("Job %s is not longer valid\n", jobId)
			delete(JobPID, jobId)
			delete(JobMetrics, jobId)
			delete(JobCache, jobId)
		}
	}
}

func getAllPids(jobId string) []uint32 {
	var pids []uint32

	jobDir := PATH + PATTERN + jobId

	err := filepath.WalkDir(jobDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "cgroups.procs" {
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

func getAllPidsFromJob(JobPID map[string][]uint32) {

	fmt.Println("Looking for PIDS")
	for jobId, _ := range JobPID {
		JobPID[jobId] = getAllPids(jobId)
		fmt.Printf("PIDS from JOB %s:\n%v ", jobId, JobPID[jobId])
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

func getCPUData(nodeName string, flag int) []JobStat {

	var jobStats []JobStat = make([]JobStat, 0)

	// key is job_id number to search for a dir need to add PATTERN
	JobCache := make(map[string]jobData)
	JobPid := make(map[string][]uint32)
	JobMetrics := make(map[string]cpuMetrics)

	fmt.Printf("Size of JobCache : %d\n", len(JobCache))

	getNewJobs(JobCache, nodeName)
	clearOldJobs(JobCache, JobPid, JobMetrics)
	//getAllPidsFromJob(JobPid) has a bug

	if flag == 1 {
		return jobStats
	}
	return nil
}

func Jobs() []JobStat {
	return getCPUData("gpunode3", 1)
}

func initCPUMonitor(nodeName string) {

	fmt.Println("Init CPU Monitoring")

	ticker := time.NewTicker(time.Duration(TimeInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		//obsługa błedów

		case t := <-ticker.C:
			{
				fmt.Printf("CPU RAM %s\n", t.Format("2006-01-02 15:04:05"))
				getCPUData(nodeName, 0)
			}
		}
	}

}
