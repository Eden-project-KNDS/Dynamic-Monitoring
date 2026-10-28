package monitor

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
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

func get_new_jobs(JobCache map[string]jobData, nodeName string) {

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

// parseScontrolValue wyciąga wartość klucza w formacie Key=Value z wyjścia scontrol
func parseScontrolValue(input, key string) string {
	re := regexp.MustCompile(fmt.Sprintf(`\b%s=([^\s]+)`, regexp.QuoteMeta(key)))
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

func initCPUMonitor(nodeName string) {

	fmt.Println("Init CPU Monitoring")

	JobCache := make(map[string]jobData)
	//JobPid := make(map[string][]int32)
	//JobMetrics := make(map[string]cpuMetrics)

	ticker := time.NewTicker(time.Duration(TimeInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		//obsługa błedów

		case t := <-ticker.C:
			{
				fmt.Printf("CPU RAM %s\n", t.Format("2006-01-02 15:04:05"))
				get_new_jobs(JobCache, nodeName)

			}
		}
	}

}
