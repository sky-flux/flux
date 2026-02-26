# flux

[English](README.md) | [简体中文](README_zh-CN.md) | [繁體中文](README_zh-TW.md) | [Deutsch](README_de.md) | [Suomi](README_fi.md) | [Español](README_es.md)

[![CI](https://img.shields.io/github/actions/workflow/status/sky-flux/flux/ci.yml?branch=main&label=CI)](https://github.com/sky-flux/flux/actions)
[![codecov](https://codecov.io/github/sky-flux/flux/graph/badge.svg?token=YT941R23LJ)](https://codecov.io/github/sky-flux/flux)
[![Go Report Card](https://goreportcard.com/badge/github.com/sky-flux/flux)](https://goreportcard.com/report/github.com/sky-flux/flux)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Reine Go-Implementierung des **FSRS v6** Spaced-Repetition-Algorithmus. Keine Abhängigkeiten außerhalb der Standardbibliothek.

flux bietet eine vollständige Planungs-Engine für Karteikarten-Anwendungen: Wiederholungsplanung, Gedächtniszustandsverfolgung, Parameteroptimierung aus dem Wiederholungsverlauf und Berechnung der optimalen Behaltenrate mittels Monte-Carlo-Simulation.

## Funktionen

- **FSRS v6 Algorithmus** -- vollständige Implementierung des neuesten Free Spaced Repetition Schedulers mit 21 trainierbaren Parametern
- **Keine Abhängigkeiten** -- nur die Go-Standardbibliothek
- **Kartenlebenszyklus-Verwaltung** -- Learning-, Review- und Relearning-Zustände mit konfigurierbaren Schrittdauern
- **Parameteroptimierer** -- Mini-Batch-Gradientenabstieg mit Adam und Cosine Annealing zum Trainieren von Parametern aus Wiederholungsprotokollen
- **Optimale Behaltenrate** -- Monte-Carlo-Simulation zur Ermittlung des Behaltenziels, das die Gesamtwiederholungskosten minimiert
- **Abrufbarkeit** -- Berechnung der Erinnerungswahrscheinlichkeit für jede Karte zu jedem Zeitpunkt
- **Vorschau & Neuplanung** -- Vorschau aller Bewertungsergebnisse vor dem Festschreiben oder Wiedergabe von Wiederholungsprotokollen zur Wiederherstellung des Kartenzustands
- **Intervall-Fuzzing** -- optionale Randomisierung zur Verteilung von Wiederholungen und Vermeidung von Häufungen
- **JSON-Serialisierung** -- Card, Rating, State, Scheduler und ReviewLog implementieren alle JSON-Marshaling
- **Deterministisch & testbar** -- Fuzzing kann für reproduzierbare Tests deaktiviert werden

## Schnellstart

```bash
go get github.com/sky-flux/flux
```

Erstelle eine Karte, überprüfe sie mehrmals und beobachte, wie sich die Planung anpasst:

```go
s, _ := flux.NewScheduler(flux.SchedulerConfig{DesiredRetention: 0.9})
card := flux.NewCard(1)
now := time.Now()

// Erste Wiederholung — Karte bewegt sich durch die Lernschritte
card, _ = s.ReviewCard(card, flux.Good, now)
fmt.Println(card.State) // Learning (noch ein Schritt)

// Zweite Wiederholung — wechselt in den Review-Zustand
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.State) // Review
fmt.Println(card.Due)   // ~2 Tage ab jetzt

// Dritte Wiederholung — Intervall wächst mit jeder erfolgreichen Erinnerung
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.Due)   // ~10 Tage ab jetzt

// Erinnerungswahrscheinlichkeit zu einem beliebigen Zeitpunkt prüfen
r := s.Retrievability(card, card.Due)
fmt.Printf("%.0f%%\n", r*100) // ~90% (entspricht DesiredRetention)

// Vorschau aller vier Bewertungsergebnisse vor dem Festschreiben
preview := s.PreviewCard(card, card.Due)
for _, rating := range []flux.Rating{flux.Again, flux.Hard, flux.Good, flux.Easy} {
    fmt.Printf("%s → %s\n", rating, preview[rating].Due)
}
```

Siehe [`examples/`](examples/) für vollständige ausführbare Programme zum grundlegenden Lebenszyklus, zur Parameteroptimierung und zur Neuplanung von Wiederholungsprotokollen.

## API-Überblick

### Kerntypen

```go
// Card enthält den Planungszustand für eine einzelne Karteikarte.
type Card struct {
    CardID     int64
    State      State      // Learning, Review oder Relearning
    Step       *int       // aktueller Lern-/Wiederholungsschritt (nil im Review-Zustand)
    Stability  *float64   // Gedächtnisstabilität in Tagen (nil vor der ersten Wiederholung)
    Difficulty *float64   // Schwierigkeit des Elements (nil vor der ersten Wiederholung)
    Due        time.Time
    LastReview *time.Time
}

// Rating repräsentiert die Erinnerungsbewertung des Benutzers.
type Rating int // Again=1, Hard=2, Good=3, Easy=4

// State repräsentiert die Lernstufe einer Karte.
type State int // Learning=1, Review=2, Relearning=3

// ReviewLog zeichnet ein einzelnes Wiederholungsereignis auf.
type ReviewLog struct {
    CardID         int64
    Rating         Rating
    ReviewDatetime time.Time
    ReviewDuration *int // Millisekunden, optional
}
```

### Scheduler

```go
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error)
```

| Methode | Beschreibung |
|---------|--------------|
| `ReviewCard(card Card, rating Rating, now time.Time) (Card, ReviewLog)` | Verarbeitet eine Wiederholung und gibt die aktualisierte Karte und das Protokoll zurück |
| `PreviewCard(card Card, now time.Time) map[Rating]Card` | Vorschau der Ergebnisse für alle vier Bewertungen |
| `RescheduleCard(card Card, logs []ReviewLog) (Card, error)` | Wiedergabe von Wiederholungsprotokollen zur Wiederherstellung des Kartenzustands |
| `Retrievability(card Card, now time.Time) float64` | Berechnung der Erinnerungswahrscheinlichkeit zu einem gegebenen Zeitpunkt |

### SchedulerConfig

```go
type SchedulerConfig struct {
    Parameters       [21]float64     // Nullwert -> DefaultParameters
    DesiredRetention float64         // Nullwert -> 0.9
    LearningSteps    []time.Duration // nil -> [1m, 10m]
    RelearningSteps  []time.Duration // nil -> [10m]
    MaximumInterval  int             // Nullwert -> 36500 Tage
    DisableFuzzing   bool            // Nullwert -> false (Fuzzing aktiviert)
}
```

### Parameters

```go
var DefaultParameters [21]float64 // FSRS v6 Standardparameter aus py-fsrs
var LowerBounds [21]float64
var UpperBounds [21]float64

func ValidateParameters(p [21]float64) error
```

## Optimierer

Das `optimizer`-Unterpaket trainiert FSRS-Parameter aus echtem Wiederholungsverlauf und berechnet optimale Behaltenziele.

```go
import "github.com/sky-flux/flux/optimizer"

// Sammle Wiederholungsprotokolle aus deiner Anwendung (z.B. aus einer Datenbank).
// Jedes Protokoll zeichnet auf, welche Karte überprüft wurde, die Bewertung und den Zeitpunkt.
logs := []flux.ReviewLog{
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day1},
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day3},
    // ... Hunderte oder Tausende echter Wiederholungen
}

opt := optimizer.NewOptimizer(optimizer.OptimizerConfig{})

// Personalisierte Parameter aus dem Wiederholungsverlauf trainieren
params, err := opt.ComputeOptimalParameters(ctx, logs)

// Die optimierten Parameter in einem neuen Scheduler verwenden
s, _ := flux.NewScheduler(flux.SchedulerConfig{Parameters: params})

// Optional: Das Behaltenziel finden, das die Gesamtwiederholungskosten minimiert.
// Erfordert, dass ReviewDuration in jedem Protokoll gesetzt ist.
retention, err := opt.ComputeOptimalRetention(ctx, params, logs)
```

### OptimizerConfig

| Feld | Standard | Beschreibung |
|------|----------|--------------|
| `Epochs` | 5 | Trainings-Epochen |
| `MiniBatchSize` | 512 | Wiederholungen pro Mini-Batch |
| `LearningRate` | 0.04 | Initiale Adam-Lernrate |
| `MaxSeqLen` | 64 | Max. Wiederholungen pro Karte |

## Leistung

Umgebung: Mac Mini (Apple M4 Pro, 64 GB RAM, 2T SSD), macOS 26.2, Go 1.26 darwin/arm64

| Benchmark | Ergebnis | Ziel |
|-----------|----------|------|
| ReviewCard | 181 ns/op, 80 B, 6 allocs | < 500 ns |
| GetRetrievability | 24 ns/op, 0 B, 0 allocs | < 100 ns |
| PreviewCard | 820 ns/op, 1112 B, 31 allocs | < 2 us |
| Optimize1000 | 0.49 s | < 2 s |
| Optimize10000 | 4.59 s | < 15 s |

## Übereinstimmung mit py-fsrs

flux ist eine zeilengetreue Portierung der Referenzimplementierung [py-fsrs](https://github.com/open-spaced-repetition/py-fsrs). Alle 21 FSRS v6 Parameter, die Gedächtniszustandsgleichungen, die Stabilitäts-/Schwierigkeitsaktualisierungsformeln und die Intervallberechnungslogik stimmen mit der Python-Referenz überein. Die Testsuite validiert die Ausgabegleichheit mit py-fsrs für dieselben Eingaben und Parametersätze.

## Beispiele

Das Verzeichnis [`examples/`](examples/) enthält vollständige ausführbare Programme:

| Beispiel | Beschreibung | Ausführen |
|----------|--------------|-----------|
| [`basic`](examples/basic/) | Kartenerstellung, Wiederholungsschleife, Vorschau | `go run ./examples/basic/` |
| [`optimizer`](examples/optimizer/) | Parametertraining, optimale Behaltenrate | `go run ./examples/optimizer/` |
| [`reschedule`](examples/reschedule/) | Wiedergabe von Wiederholungsprotokollen zur Zustandswiederherstellung | `go run ./examples/reschedule/` |

## Mitwirken

Siehe [CONTRIBUTING.md](CONTRIBUTING.md) für Richtlinien zur Mitarbeit an diesem Projekt.

## Lizenz

[MIT](LICENSE)
