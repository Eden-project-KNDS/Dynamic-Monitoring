# Eden Job Exporter: specyfikacja architektoniczna (wersja robocza)

## 1. Cel

Demon działający stale na każdym węźle obliczeniowym. Bez żadnych ręcznych kroków udostępnia dla **każdego joba Slurma**:

- ile zasobów (CPU, RAM, GPU) job **zużywa**,
- ile zasobów job **zarezerwował**.

Na tej podstawie Prometheus i Grafana pokazują, które joby marnują zasoby.

---

## 2. Architektura

```mermaid
flowchart LR
    subgraph NODE["Węzeł obliczeniowy (każdy)"]
        direction TB
        SD["systemd"] -->|"start, restart"| EXP["eden-job-exporter<br/>/metrics"]
        CG[("cgroup joba<br/>(liczniki jądra)")]
        NV[("NVML<br/>(sterownik NVIDIA)")]
        J1["job 1001"] -. "procesy w" .-> CG
        J2["job 1002"] -. "procesy w" .-> CG
        EXP -->|"zużycie CPU / RAM"| CG
        EXP -->|"zużycie GPU"| NV
    end

    CTL["slurmctld"]
    EXP -->|"opis i rezerwacja joba<br/>(raz na job)"| CTL

    PROM["Prometheus"] -->|"pobiera /metrics<br/>co 15–30 s"| EXP
    GRAF["Grafana / raporty"] --> PROM
```

- **Model pull:** demon niczego nie wysyła sam. Prometheus przychodzi po dane.
- **Zużycie** pochodzi z węzła (cgroup, NVML), **opis i rezerwacja** joba ze Slurma.

---

## 3. Cykl życia

```mermaid
stateDiagram-v2
    [*] --> Start
    Start --> GPU: sterownik NVIDIA dostępny
    Start --> CPUonly: brak kart / sterownika
    GPU --> Działa
    CPUonly --> Działa
    Działa --> Działa: odpowiada Prometheusowi
    Działa --> Zamykanie: SIGTERM / SIGINT
    Zamykanie --> [*]: dokończ odpowiedzi,<br/>zwolnij NVML
```

- Uruchamiany przez **systemd**: start z systemem (po Slurmie), restart po awarii, bez uprawnień roota.
- **Jedna instancja na węźle:** zapewnia ją systemd.
- Węzeł **bez GPU**: demon działa dalej i zbiera tylko CPU i RAM.

---

## 4. Odkrywanie jobów

```mermaid
flowchart TD
    T["co ~10 s, w tle"] --> L["sprawdź, jakie joby są na węźle"]
    L --> C{"job znany?"}
    C -->|tak| K["nic nie rób"]
    C -->|nie| S["pobierz opis ze Slurma:<br/>użytkownik, partycja, rezerwacja CPU/RAM/GPU"]
    S --> P["zapamiętaj w liście jobów"]
    L --> E{"job zniknął?"}
    E -->|tak| D["usuń z listy"]
```

- Odkrywanie działa w tle, niezależnie od Prometheusa, bo zapytanie do Slurma bywa wolne.
- Slurm jest pytany **raz na job**, a nie przy każdym pomiarze.

---

## 5. Przypisanie GPU do joba

```mermaid
flowchart LR
    subgraph ALOK["Przydział ze Slurma"]
        A1["job 1001 → GPU 0, 1"]
        A2["job 1002 → GPU 2"]
    end
    subgraph POM["Pomiar kart (NVML)"]
        G0["GPU 0: 95%"]
        G1["GPU 1: 0%  ← zmarnowana"]
        G2["GPU 2: 40%"]
        G3["GPU 3: 0%  ← wolna"]
    end
    A1 --> G0
    A1 --> G1
    A2 --> G2
```

Kartę przypisujemy do joba **według przydziału Slurma**, a nie według procesów działających na karcie.
Karta zarezerwowana, ale nieużywana, nie ma procesów, więc tylko przydział pozwala odróżnić kartę *zmarnowaną* od *wolnej*.

---

## 6. Pobieranie danych przez Prometheusa

```mermaid
sequenceDiagram
    participant P as Prometheus
    participant E as eden-job-exporter
    participant L as lista jobów
    participant N as cgroup + NVML

    P->>E: GET /metrics
    E->>L: jakie joby są teraz?
    loop każdy job i każda karta
        E->>N: odczytaj bieżące zużycie
    end
    E-->>P: metryki (zużycie + rezerwacja)
```

- Dane odczytywane są **w chwili zapytania**, demon nie ma własnej pętli pomiarowej.
- Czas CPU eksportowany jest jako rosnący licznik, obciążenie liczy Prometheus.

---

## 7. Metryki

To „umowa” z Grafaną i raporty: nazwy zmieniamy tylko świadomie.

| Metryka | Co mierzy |
|---|---|
| `eden_job_info` | opis joba: użytkownik, konto, partycja, nazwa |
| `eden_job_cpu_usage_seconds_total` | zużyty czas CPU (licznik) |
| `eden_job_cpus_allocated` | zarezerwowane rdzenie |
| `eden_job_memory_used_bytes` | bieżące zużycie RAM |
| `eden_job_memory_peak_bytes` | szczytowe zużycie RAM |
| `eden_job_memory_allocated_bytes` | zarezerwowany RAM |
| `eden_job_gpus_allocated` | zarezerwowane karty |
| `eden_job_time_limit_seconds` | limit czasu joba |
| `eden_job_start_time_seconds` | czas startu joba |
| `eden_gpu_utilization_ratio` | wykorzystanie karty (0–1), z przypisaniem do joba |
| `eden_gpu_memory_used_bytes` | zajęta pamięć karty |

Przykład: GPU zarezerwowane, ale bezczynne od 30 minut:

```promql
avg_over_time(eden_gpu_utilization_ratio{job_id!=""}[30m]) < 0.05
```

---
