package monitor

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

func checkHardware(t *testing.T) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		t.Skipf("No nvidia devices: %v", nvml.ErrorString(ret))
	}
	nvml.Shutdown()
}

func TestNewGPUMonitor(t *testing.T) {
	checkHardware(t)
	monitor := NewGPUMonitor()
	if monitor == nil {
		t.Fatal("monitor == nil\n")
	}
	defer monitor.Close()

	if monitor.gpuCount < 0 {
		t.Errorf("Gpu count can't be a negative number : %v\n", monitor.gpuCount)
	}
	if monitor.gpuCount > 0 && len(monitor.devices) != monitor.gpuCount {
		t.Errorf("Expected %d devices, got %d\n", monitor.gpuCount, len(monitor.devices))
	}
}

func TestGpuGetStats(t *testing.T) {
	checkHardware(t)
	monitor := NewGPUMonitor()
	defer monitor.Close()

	if monitor.gpuCount < 0 {
		t.Skip("No gpu to test\n")
	}
	stats := monitor.getGpuStats(1)

	if len(*stats) != monitor.gpuCount {
		t.Errorf("Expected %d stats, recieved %d", monitor.gpuCount, len(*stats))
	}

	if len(*stats) > 0 {
		if (*stats)[0].UUID == "" {
			t.Error("Expected UUID gpu, got empty string")
		}
	}

}

func TestParseCgroupReader(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Slurm cgroups v2 - standardowy format z job_12345",
			input: "0::/system.slice/slurmstepd.scope/job_12345/step_0\n",
			want:  "job_12345",
		},
		{
			name:  "Slurm cgroups v1 - podkatalog z job_654321",
			input: "12:memory:/slurm/uid_1000/job_654321/step_interactive\n11:devices:/slurm/uid_1000/job_654321\n",
			want:  "job_654321",
		},
		{
			name:  "Segment dokładnie o długości 4 znaków (job1)",
			input: "0::/user.slice/job1/test\n",
			want:  "job1",
		},
		{
			name:  "Nazwa 'job' bez dodatkowych znaków (len <= 3) - powinna być zignorowana",
			input: "0::/system.slice/job/step_0\n",
			want:  "",
		},
		{
			name:  "Brak segmentu zaczynającego się od 'job'",
			input: "0::/user.slice/user-1000.slice/session-2.scope\n",
			want:  "",
		},
		{
			name:  "Pusta zawartość pliku",
			input: "",
			want:  "",
		},
		{
			name:  "Dopasowanie w drugiej linii",
			input: "1:name=systemd:/\n0::/system.slice/slurmstepd.scope/job_99999\n",
			want:  "job_99999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			got := parseCgroupReader(r)
			if got != tt.want {
				t.Errorf("parseCgroupReader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseProcCgroups_Integration(t *testing.T) {
	t.Run("Nieistniejący PID zwraca pusty ciąg", func(t *testing.T) {
		// Bardzo wysoki PID, który nie istnieje w systemie
		got := parseProcCgroups(99999999)
		if got != "" {
			t.Errorf("ParseProcCgroups(99999999) = %q, want %q", got, "")
		}
	})

	t.Run("Test z atrapą pliku /proc w katalogu tymczasowym", func(t *testing.T) {
		// Tworzymy tymczasową strukturę plików udającą /proc/[PID]/cgroup
		tempDir := t.TempDir()
		fakePid := uint32(12345)
		pidDir := filepath.Join(tempDir, strconv.FormatUint(uint64(fakePid), 10))

		err := os.MkdirAll(pidDir, 0755)
		if err != nil {
			t.Fatalf("Nie udało się utworzyć katalogu: %v", err)
		}

		cgroupFile := filepath.Join(pidDir, "cgroup")
		content := "0::/system.slice/slurmstepd.scope/job_777777/step_0\n"
		err = os.WriteFile(cgroupFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Nie udało się zapisać pliku cgroup: %v", err)
		}

		// Testujemy bezpośrednie czytanie utworzonego pliku przez parser
		f, err := os.Open(cgroupFile)
		if err != nil {
			t.Fatalf("Nie udało się otworzyć pliku: %v", err)
		}
		defer f.Close()

		got := parseCgroupReader(f)
		want := "job_777777"
		if got != want {
			t.Errorf("parseCgroupReader() z pliku tymczasowego = %q, want %q", got, want)
		}
	})
}
