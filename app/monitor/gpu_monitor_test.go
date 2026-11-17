package monitor

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

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
	stats := monitor.getGpuStats()

	if len(stats) != monitor.gpuCount {
		t.Errorf("Expected %d stats, received %d", monitor.gpuCount, len(stats))
	}

	if len(stats) > 0 {
		if stats[0].UUID == "" {
			t.Error("Expected UUID gpu, got empty string")
		}
	}
}

func TestTryParseCgroupReader(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "Slurm cgroups v2 - standardowy format z job_12345",
			input: "0::/system.slice/slurmstepd.scope/job_12345/step_0\n",
			want:  "12345",
		},
		{
			name:  "Slurm cgroups v1 - podkatalog z job_654321",
			input: "12:memory:/slurm/uid_1000/job_654321/step_interactive\n11:devices:/slurm/uid_1000/job_654321\n",
			want:  "654321",
		},
		{
			name:  "Segment bez podkreślnika (job1) to nie katalog joba Slurma",
			input: "0::/user.slice/job1/test\n",
			want:  "",
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
			want:  "99999",
		},
		{
			name:    "Pusty numer joba",
			input:   "0::/slurm/job_/step_0\n",
			wantErr: true,
		},
		{
			name:    "Numer joba nie jest liczbą",
			input:   "0::/slurm/job_abc/step_0\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			got, err := tryParseCgroupReader(r)
			if (err != nil) != tt.wantErr {
				t.Fatalf("tryParseCgroupReader() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("tryParseCgroupReader() = %q, want %q", got, tt.want)
			}
		})
	}

	wantErr := errors.New("reader failed")
	got, err := tryParseCgroupReader(iotest.ErrReader(wantErr))
	if got != "" || !errors.Is(err, wantErr) {
		t.Errorf("tryParseCgroupReader() = %q, %v, want empty result and %v", got, err, wantErr)
	}
}

func TestTryParseProcCgroups_Integration(t *testing.T) {
	t.Run("Nieistniejący PID zwraca pusty ciąg", func(t *testing.T) {
		// Bardzo wysoki PID, który nie istnieje w systemie
		got, err := tryParseProcCgroups(99999999)
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("tryParseProcCgroups(99999999) error = %v, want file not found", err)
		}
		if got != "" {
			t.Errorf("tryParseProcCgroups(99999999) = %q, want %q", got, "")
		}
	})

	t.Run("Test z atrapą pliku /proc w katalogu tymczasowym", func(t *testing.T) {
		// Tworzymy tymczasową strukturę plików udającą /proc/[PID]/cgroup
		tempDir := t.TempDir()
		fakePid := uint32(12345)
		pidDir := filepath.Join(tempDir, strconv.FormatUint(uint64(fakePid), 10))

		err := os.MkdirAll(pidDir, 0o755)
		if err != nil {
			t.Fatalf("Nie udało się utworzyć katalogu: %v", err)
		}

		cgroupFile := filepath.Join(pidDir, "cgroup")
		content := "0::/system.slice/slurmstepd.scope/job_777777/step_0\n"
		err = os.WriteFile(cgroupFile, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("Nie udało się zapisać pliku cgroup: %v", err)
		}

		// Testujemy bezpośrednie czytanie utworzonego pliku przez parser
		f, err := os.Open(cgroupFile)
		if err != nil {
			t.Fatalf("Nie udało się otworzyć pliku: %v", err)
		}
		defer f.Close()

		got, err := tryParseCgroupReader(f)
		if err != nil {
			t.Fatalf("tryParseCgroupReader() error = %v", err)
		}
		want := "777777"
		if got != want {
			t.Errorf("tryParseCgroupReader() z pliku tymczasowego = %q, want %q", got, want)
		}
	})
}

type gpuTestDevice struct {
	nvml.Device
	util      nvml.Utilization
	utilRet   nvml.Return
	memory    nvml.Memory_v2
	memoryRet nvml.Return
	otherRet  nvml.Return
}

func (d *gpuTestDevice) GetUUID() (string, nvml.Return) {
	return "GPU-test", nvml.SUCCESS
}

func (d *gpuTestDevice) GetUtilizationRates() (nvml.Utilization, nvml.Return) {
	return d.util, d.utilRet
}

func (d *gpuTestDevice) GetMemoryInfo_v2() (nvml.Memory_v2, nvml.Return) {
	return d.memory, d.memoryRet
}

func (d *gpuTestDevice) GetTemperature(nvml.TemperatureSensors) (uint32, nvml.Return) {
	return 0, d.otherRet
}

func (d *gpuTestDevice) GetPowerUsage() (uint32, nvml.Return) {
	return 0, d.otherRet
}

func (d *gpuTestDevice) GetComputeRunningProcesses() ([]nvml.ProcessInfo, nvml.Return) {
	return nil, d.otherRet
}

func TestGpuGetStatsReadFailures(t *testing.T) {
	device := &gpuTestDevice{
		util:   nvml.Utilization{Gpu: 75},
		memory: nvml.Memory_v2{Used: 64 * 1024 * 1024},
	}
	monitor := gpuMonitor{1, []nvml.Device{device}, make([]gpuMetrics, 1)}
	want := GPUStat{UUID: "GPU-test", UtilPercent: 75, UtilPercentValid: true, MemoryUsed: 64 * 1024 * 1024, MemoryUsedValid: true}
	if stats := monitor.getGpuStats(); len(stats) != 1 || stats[0] != want {
		t.Fatalf("Successful sample = %+v, want %+v", stats, want)
	}

	device.utilRet = nvml.ERROR_UNKNOWN
	device.otherRet = nvml.ERROR_NOT_SUPPORTED
	device.memory.Used = 32 * 1024 * 1024
	want = GPUStat{UUID: "GPU-test", MemoryUsed: 32 * 1024 * 1024, MemoryUsedValid: true}
	if stats := monitor.getGpuStats(); len(stats) != 1 || stats[0] != want {
		t.Fatalf("Utilization failure = %+v, want %+v", stats, want)
	}

	device.utilRet = nvml.SUCCESS
	device.util.Gpu = 25
	device.memoryRet = nvml.ERROR_UNKNOWN
	want = GPUStat{UUID: "GPU-test", UtilPercent: 25, UtilPercentValid: true}
	if stats := monitor.getGpuStats(); len(stats) != 1 || stats[0] != want {
		t.Errorf("Memory failure = %+v, want %+v", stats, want)
	}
}
