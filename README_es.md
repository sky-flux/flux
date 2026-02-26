# flux

[English](README.md) | [简体中文](README_zh-CN.md) | [繁體中文](README_zh-TW.md) | [Deutsch](README_de.md) | [Suomi](README_fi.md) | [Español](README_es.md)

[![CI](https://img.shields.io/github/actions/workflow/status/sky-flux/flux/ci.yml?branch=main&label=CI)](https://github.com/sky-flux/flux/actions)
[![codecov](https://codecov.io/github/sky-flux/flux/graph/badge.svg?token=YT941R23LJ)](https://codecov.io/github/sky-flux/flux)
[![Go Report Card](https://goreportcard.com/badge/github.com/sky-flux/flux)](https://goreportcard.com/report/github.com/sky-flux/flux)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Implementación en Go puro del algoritmo de repetición espaciada **FSRS v6**. Sin dependencias fuera de la biblioteca estándar.

flux proporciona un motor de planificación completo para aplicaciones de tarjetas didácticas: planificación de repasos, seguimiento del estado de la memoria, optimización de parámetros a partir del historial de repasos y cálculo de la retención óptima mediante simulación de Monte Carlo.

## Características

- **Algoritmo FSRS v6** -- implementación completa del último Free Spaced Repetition Scheduler con 21 parámetros entrenables
- **Sin dependencias** -- solo la biblioteca estándar de Go
- **Gestión del ciclo de vida de tarjetas** -- estados Learning, Review y Relearning con duraciones de pasos configurables
- **Optimizador de parámetros** -- descenso de gradiente por mini-lotes con Adam y recocido coseno para entrenar parámetros a partir de registros de repaso
- **Retención óptima** -- simulación de Monte Carlo para encontrar el objetivo de retención que minimiza el costo total de repaso
- **Recuperabilidad** -- cálculo de la probabilidad de recuerdo para cualquier tarjeta en cualquier momento
- **Vista previa y reprogramación** -- vista previa de todos los resultados de calificación antes de confirmar, o reproducción de registros de repaso para reconstruir el estado de la tarjeta
- **Difuminado de intervalos** -- aleatorización opcional para distribuir repasos y evitar agrupamiento
- **Serialización JSON** -- Card, Rating, State, Scheduler y ReviewLog implementan serialización JSON
- **Determinista y testeable** -- el difuminado se puede desactivar para pruebas reproducibles

## Inicio rápido

```bash
go get github.com/sky-flux/flux
```

Crea una tarjeta, repásala varias veces y observa cómo se adapta la planificación:

```go
s, _ := flux.NewScheduler(flux.SchedulerConfig{DesiredRetention: 0.9})
card := flux.NewCard(1)
now := time.Now()

// Primer repaso — la tarjeta avanza por los pasos de aprendizaje
card, _ = s.ReviewCard(card, flux.Good, now)
fmt.Println(card.State) // Learning (un paso más)

// Segundo repaso — gradúa al estado Review
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.State) // Review
fmt.Println(card.Due)   // ~2 días a partir de ahora

// Tercer repaso — el intervalo crece con cada recuerdo exitoso
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.Due)   // ~10 días a partir de ahora

// Consultar la probabilidad de recuerdo en cualquier momento
r := s.Retrievability(card, card.Due)
fmt.Printf("%.0f%%\n", r*100) // ~90% (coincide con DesiredRetention)

// Vista previa de los cuatro resultados de calificación antes de confirmar
preview := s.PreviewCard(card, card.Due)
for _, rating := range []flux.Rating{flux.Again, flux.Hard, flux.Good, flux.Easy} {
    fmt.Printf("%s → %s\n", rating, preview[rating].Due)
}
```

Consulta [`examples/`](examples/) para ver programas completos ejecutables que cubren el ciclo de vida básico, la optimización de parámetros y la reprogramación de registros de repaso.

## Resumen de la API

### Tipos principales

```go
// Card contiene el estado de planificación de una tarjeta didáctica.
type Card struct {
    CardID     int64
    State      State      // Learning, Review o Relearning
    Step       *int       // paso actual de aprendizaje/reaprendizaje (nil en Review)
    Stability  *float64   // estabilidad de la memoria en días (nil antes del primer repaso)
    Difficulty *float64   // dificultad del elemento (nil antes del primer repaso)
    Due        time.Time
    LastReview *time.Time
}

// Rating representa la evaluación de recuerdo del usuario.
type Rating int // Again=1, Hard=2, Good=3, Easy=4

// State representa la etapa de aprendizaje de una tarjeta.
type State int // Learning=1, Review=2, Relearning=3

// ReviewLog registra un único evento de repaso.
type ReviewLog struct {
    CardID         int64
    Rating         Rating
    ReviewDatetime time.Time
    ReviewDuration *int // milisegundos, opcional
}
```

### Scheduler

```go
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error)
```

| Método | Descripción |
|--------|-------------|
| `ReviewCard(card Card, rating Rating, now time.Time) (Card, ReviewLog)` | Procesa un repaso y devuelve la tarjeta actualizada y el registro |
| `PreviewCard(card Card, now time.Time) map[Rating]Card` | Vista previa de los resultados para las cuatro calificaciones |
| `RescheduleCard(card Card, logs []ReviewLog) (Card, error)` | Reproduce registros de repaso para reconstruir el estado de la tarjeta |
| `Retrievability(card Card, now time.Time) float64` | Calcula la probabilidad de recuerdo en un momento dado |

### SchedulerConfig

```go
type SchedulerConfig struct {
    Parameters       [21]float64     // cero -> DefaultParameters
    DesiredRetention float64         // cero -> 0.9
    LearningSteps    []time.Duration // nil -> [1m, 10m]
    RelearningSteps  []time.Duration // nil -> [10m]
    MaximumInterval  int             // cero -> 36500 días
    DisableFuzzing   bool            // cero -> false (difuminado activado)
}
```

### Parameters

```go
var DefaultParameters [21]float64 // Parámetros predeterminados de FSRS v6 de py-fsrs
var LowerBounds [21]float64
var UpperBounds [21]float64

func ValidateParameters(p [21]float64) error
```

## Optimizador

El subpaquete `optimizer` entrena parámetros FSRS a partir del historial real de repasos y calcula objetivos de retención óptimos.

```go
import "github.com/sky-flux/flux/optimizer"

// Recopila registros de repaso de tu aplicación (por ejemplo, de una base de datos).
// Cada registro indica qué tarjeta fue repasada, la calificación y cuándo.
logs := []flux.ReviewLog{
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day1},
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day3},
    // ... cientos o miles de repasos reales
}

opt := optimizer.NewOptimizer(optimizer.OptimizerConfig{})

// Entrenar parámetros personalizados a partir del historial de repasos
params, err := opt.ComputeOptimalParameters(ctx, logs)

// Usar los parámetros optimizados en un nuevo planificador
s, _ := flux.NewScheduler(flux.SchedulerConfig{Parameters: params})

// Opcional: encontrar el objetivo de retención que minimiza el costo total de repaso.
// Requiere que ReviewDuration esté establecido en cada registro.
retention, err := opt.ComputeOptimalRetention(ctx, params, logs)
```

### OptimizerConfig

| Campo | Predeterminado | Descripción |
|-------|----------------|-------------|
| `Epochs` | 5 | Épocas de entrenamiento |
| `MiniBatchSize` | 512 | Repasos por mini-lote |
| `LearningRate` | 0.04 | Tasa de aprendizaje inicial de Adam |
| `MaxSeqLen` | 64 | Máx. repasos por tarjeta |

## Rendimiento

Entorno: Mac Mini (Apple M4 Pro, 64 GB RAM, 2T SSD), macOS 26.2, Go 1.26 darwin/arm64

| Benchmark | Resultado | Objetivo |
|-----------|-----------|----------|
| ReviewCard | 181 ns/op, 80 B, 6 allocs | < 500 ns |
| GetRetrievability | 24 ns/op, 0 B, 0 allocs | < 100 ns |
| PreviewCard | 820 ns/op, 1112 B, 31 allocs | < 2 us |
| Optimize1000 | 0.49 s | < 2 s |
| Optimize10000 | 4.59 s | < 15 s |

## Alineación con py-fsrs

flux es una portación línea por línea de la implementación de referencia [py-fsrs](https://github.com/open-spaced-repetition/py-fsrs). Los 21 parámetros de FSRS v6, las ecuaciones de estado de la memoria, las fórmulas de actualización de estabilidad/dificultad y la lógica de cálculo de intervalos coinciden con la referencia en Python. La suite de pruebas valida la paridad de resultados con py-fsrs para las mismas entradas y conjuntos de parámetros.

## Ejemplos

El directorio [`examples/`](examples/) contiene programas completos ejecutables:

| Ejemplo | Descripción | Ejecutar |
|---------|-------------|----------|
| [`basic`](examples/basic/) | Creación de tarjetas, bucle de repaso, vista previa | `go run ./examples/basic/` |
| [`optimizer`](examples/optimizer/) | Entrenamiento de parámetros, retención óptima | `go run ./examples/optimizer/` |
| [`reschedule`](examples/reschedule/) | Reproducción de registros de repaso para reconstruir estado | `go run ./examples/reschedule/` |

## Contribuir

Consulta [CONTRIBUTING.md](CONTRIBUTING.md) para las directrices sobre cómo contribuir a este proyecto.

## Licencia

[MIT](LICENSE)
