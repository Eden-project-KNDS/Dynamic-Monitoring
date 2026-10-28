package monitor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

func get_new_jobs(JobCache map[string]jobData) {

	entries, err := os.ReadDir(PATH)
	if err != nil {
		log.Fatalln("Something went wrong when reading cgroups")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), PATTERN) && !keyExists(JobCache, entry.Name()) {
			filepath.Join(PATH, entry.Name())

		}
	}

}
func initCPUMonitor() {

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
				get_new_jobs(JobCache)

			}
		}
	}

}
