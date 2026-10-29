# Dokumentacja aktualnego kodu

## Zakres

Ten dokument opisuje implementację znajdującą się w katalogu `app/`. Jest to wczesny prototyp monitora węzła obliczeniowego Slurma: odczytuje wybrane parametry kart NVIDIA oraz zawiera rozpoczęty mechanizm wykrywania katalogów cgroups jobów. Nie jest to jeszcze kompletny exporter metryk ani implementacja architektury opisanej w [specyfikacji Eden Job Exporter](eden-job-exporter-spec.md).

## Struktura aplikacji

- `main.go` — obsługuje argument wiersza poleceń i uruchamia monitoring.
- `monitor/monitor.go` — uruchamia osobne goroutines dla monitoringu GPU i CPU/RAM; określa wspólny interwał próbkowania.
- `monitor/gpu_monitor.go` — inicjalizuje NVML, wykrywa urządzenia GPU i cyklicznie odczytuje ich parametry.
- `monitor/cgroups_monitor.go` — zawiera szkic wykrywania jobów w cgroups, strukturę ich danych i strukturę planowanych metryk CPU/RAM.
- `Makefile` — definiuje formatowanie, lint, testy i budowanie binarium.
- `go.mod` — definiuje moduł Go `slurm-monitor` i zależność `github.com/NVIDIA/go-nvml`.

## Uruchomienie

Program oczekuje dokładnie jednego argumentu: nazwy węzła. Przy innej liczbie argumentów wyświetla krótką instrukcję użycia i kończy działanie.

```bash
cd app
make build
./slurm-cgroup-monitor <nazwa_węzła>
```

Argument nazwy węzła jest przekazywany do obu monitorów, ale w obecnym kodzie nie wpływa na odczyt danych ani nie jest dołączany do wyjścia. Po uruchomieniu program wypisuje `Start`, a następnie dwa niezależne procesy monitorujące działają w pętli.

### Cele Makefile

- `make` (cel domyślny) uruchamia `fmt`, a następnie `lint`.
- `make fmt` wykonuje `go fmt ./...` i `goimports -w .`.
- `make lint` uruchamia `golangci-lint run`.
- `make test` uruchamia `go test -v ./...`.
- `make build` buduje binarium `slurm-cgroup-monitor`, ustawiając `CGO_CFLAGS="-Wno-deprecated-declarations"`.

## Wspólny cykl próbkowania

Stała `TimeInterval` w `monitor/monitor.go` wynosi 5 sekund. Każdy monitor tworzy własny ticker i samodzielnie wypisuje wyniki na standardowe wyjście. Nie ma centralnego harmonogramu ani wspólnego rekordu pomiarowego.

`StartMonitoring` uruchamia funkcje `initGPUMonitoring` i `initCPUMonitor` jako goroutines. Nie czeka na ich zakończenie. Monitor CPU/RAM działa w nieskończonej pętli; monitor GPU również działa w pętli, jeśli wykryje urządzenia (przy zerowej liczbie GPU kończy pracę). Kod nie obsługuje sygnałów zakończenia ani kontrolowanego zamykania.

## Monitoring GPU

Implementacja w `monitor/gpu_monitor.go` korzysta z biblioteki NVIDIA NVML przez `github.com/NVIDIA/go-nvml`.

1. Inicjalizuje NVML.
2. Odczytuje liczbę urządzeń i pobiera uchwyt do każdego GPU.
3. Co 5 sekund dla każdego urządzenia próbuje pobrać wykorzystanie GPU i pamięci, zajętą i całkowitą pamięć, temperaturę oraz pobór mocy.
4. Wypisuje odczytane wartości na standardowe wyjście.

Pamięć jest przeliczana z bajtów na MiB, a moc z miliwatów na waty. Struktura `gpuMetrics` przechowuje:

| Pole | Znaczenie |
|---|---|
| `coreUtilPercent` | Wykorzystanie GPU w procentach |
| `memoryUtilPercent` | Wykorzystanie kontrolera pamięci w procentach |
| `tempCelsius` | Temperatura GPU w stopniach Celsjusza |
| `memoryUsedMB` | Zajęta pamięć GPU w MiB |
| `memoryTotalMB` | Całkowita pamięć GPU w MiB |
| `powerWatts` | Pobór mocy w watach |

Jeśli NVML nie może się zainicjalizować, odczytać liczby urządzeń lub uzyskać uchwytu urządzenia, aplikacja kończy działanie przez `log.Fatalf`. Błędy pojedynczych odczytów w trakcie próbkowania nie są raportowane; dla danego pola pozostaje poprzednia wartość (początkowo zero). Gdy nie ma urządzeń GPU, funkcja monitorująca GPU wraca, ale monitoring CPU/RAM pozostaje uruchomiony.

Wartości nie są przypisywane do jobów Slurma ani PID-ów. `nodeName` nie jest obecnie używany w tym monitorze. Wywołanie `nvml.Shutdown` znajduje się w `defer`, ale przy niekończącej się pętli próbkowania nie zostanie osiągnięte podczas normalnej pracy.

## Monitoring cgroups i CPU/RAM

Implementacja w `monitor/cgroups_monitor.go` jest nieukończona. Korzysta ze stałych:

- ścieżka cgroups: `/sys/fs/cgroup/system.slice/slurmstepd.scope/`;
- prefiks nazw katalogów jobów: `job_`.

Funkcja `initCPUMonitor` tworzy mapę `JobCache` i co 5 sekund wywołuje `get_new_jobs`. Ta funkcja przegląda katalog cgroups i dla nowych katalogów o nazwie zaczynającej się od `job_` próbuje uruchomić `scontrol show job <nazwa_katalogu>`. Dane z odpowiedzi Slurma nie są jeszcze parsowane ani zapisywane do cache.

`parseScontrolValue` jest pomocniczym parserem wartości w formacie `Klucz=wartość`: zwraca tekst do najbliższego białego znaku lub `unknown`, jeśli klucz nie wystąpi. Nie jest jeszcze używany przez pozostały kod.

Struktura `cpuMetrics` wskazuje planowany zakres odczytów:

| Grupa | Pola |
|---|---|
| RAM | bieżące zużycie, pamięć anonimowa, szczytowe zużycie, limit i liczba zdarzeń OOM |
| CPU | łączny czas CPU, czas użytkownika, czas systemowy i czas throttlingu |

W obecnej implementacji te wartości nie są odczytywane z plików cgroups, nie są liczone ani wypisywane. Nie ma też mapowania jobów na procesy/PID-y.

Jeśli odczyt katalogu cgroups lub polecenie `scontrol` zwróci błąd, użyte `log.Fatalln`/`log.Fatalf` kończy cały proces, zamiast pominąć pojedynczy pomiar.

## Wyjście i przechowywanie danych

Jedynym obecnym wyjściem pomiarowym jest standardowe wyjście procesu. Monitor GPU wypisuje znacznik czasu oraz wartości dla kolejnych urządzeń, a monitor CPU/RAM wypisuje znacznik czasu z etykietą `CPU RAM`. Kod nie udostępnia endpointu HTTP, formatu Prometheus, plików logów, bazy danych ani interfejsu do odczytu zebranych wartości.

## Zależności i wymagania środowiskowe

- Wersja Go zadeklarowana w `app/go.mod`: `1.27.2`.
- Zależność Go: `github.com/NVIDIA/go-nvml` w wersji `v0.13.4-1`.
- Działający sterownik NVIDIA/NVML jest wymagany do pomiaru GPU.
- Do wykrywania jobów przewidziane jest polecenie Slurma `scontrol` oraz dostęp do ścieżki cgroups skonfigurowanej w kodzie.
- Cele `fmt` i `lint` wymagają odpowiednio `goimports` i `golangci-lint` dostępnych w środowisku.

## Aktualne ograniczenia

- Kod monitoringu cgroups nie jest ukończony. W `get_new_jobs` pozostawiono odwołanie `jobdata` bez definicji, a funkcja nie uzupełnia `JobCache`; projekt wymaga dokończenia tej części przed uzyskaniem działającego mapowania jobów.
- Metryki GPU są zbierane dla urządzeń, nie dla jobów; mapowanie GPU/job/PID nie jest zaimplementowane.
- Parametr węzła jest wymagany przez CLI, ale nie jest wykorzystywany do identyfikacji węzła w wynikach.
- Brakuje obsługi sygnałów i kontrolowanego zakończenia procesu.
- Błędy odczytu pojedynczych metryk GPU są pomijane bez komunikatu.
- W aktualnym środowisku polecenie `make` dochodzi do `golangci-lint`, który nie może obsłużyć zadeklarowanej wersji Go: użyty linter został zbudowany dla Go 1.23, a projekt deklaruje Go 1.27.2. To ograniczenie narzędzia lintującego; nie jest potwierdzeniem poprawnej kompilacji aplikacji.

Opis przyszłego modelu eksportu metryk, pracy Prometheusa, integracji z systemd oraz docelowego cyklu życia znajduje się w [specyfikacji architektonicznej](eden-job-exporter-spec.md). Są to założenia projektowe i nie należy ich traktować jako funkcji obecnego kodu.