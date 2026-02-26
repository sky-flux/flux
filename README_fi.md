# flux

[English](README.md) | [简体中文](README_zh-CN.md) | [繁體中文](README_zh-TW.md) | [Deutsch](README_de.md) | [Suomi](README_fi.md) | [Español](README_es.md)

[![CI](https://img.shields.io/github/actions/workflow/status/sky-flux/flux/ci.yml?branch=main&label=CI)](https://github.com/sky-flux/flux/actions)
[![codecov](https://codecov.io/github/sky-flux/flux/graph/badge.svg?token=YT941R23LJ)](https://codecov.io/github/sky-flux/flux)
[![Go Report Card](https://goreportcard.com/badge/github.com/sky-flux/flux)](https://goreportcard.com/report/github.com/sky-flux/flux)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Puhdas Go-toteutus **FSRS v6** -hajautetun kertauksen algoritmista. Ei ulkoisia riippuvuuksia standardikirjaston ulkopuolelta.

flux tarjoaa täydellisen ajoitusmoottorin muistikorttisovelluksille: kertausajoitus, muistitilan seuranta, parametrien optimointi kertaushistoriasta sekä optimaalisen muistamisprosentin laskenta Monte Carlo -simulaatiolla.

## Ominaisuudet

- **FSRS v6 -algoritmi** -- täydellinen toteutus uusimmasta Free Spaced Repetition Scheduler -algoritmista, jossa on 21 koulutettavaa parametria
- **Ei riippuvuuksia** -- vain Go:n standardikirjasto
- **Kortin elinkaaren hallinta** -- Learning-, Review- ja Relearning-tilat konfiguroitavilla vaihekestoilla
- **Parametrioptimoija** -- minierägradienttimenetelmä Adam-optimoijalla ja kosinijäähdytyksellä parametrien kouluttamiseen kertauslokista
- **Optimaalinen muistamisprosentti** -- Monte Carlo -simulaatio kokonaiskertauskustannukset minimoivan muistamisprosentin löytämiseksi
- **Palautettavuus** -- muistamistodennäköisyyden laskenta mille tahansa kortille minä tahansa ajanhetkenä
- **Esikatselu ja uudelleenajoitus** -- kaikkien arviointitulosten esikatselu ennen vahvistamista tai kertauslokien toistaminen kortin tilan uudelleenrakentamiseksi
- **Aikavälin hajautus** -- valinnainen satunnaistaminen kertausten jakamiseksi ja kasautumisen välttämiseksi
- **JSON-serialisointi** -- Card, Rating, State, Scheduler ja ReviewLog toteuttavat kaikki JSON-marshaling-rajapinnan
- **Deterministinen ja testattava** -- hajautus voidaan poistaa käytöstä toistettavia testejä varten

## Pika-aloitus

```bash
go get github.com/sky-flux/flux
```

Luo kortti, kertaa se useita kertoja ja seuraa ajoituksen mukautumista:

```go
s, _ := flux.NewScheduler(flux.SchedulerConfig{DesiredRetention: 0.9})
card := flux.NewCard(1)
now := time.Now()

// Ensimmäinen kertaus — kortti etenee oppimisvaiheiden läpi
card, _ = s.ReviewCard(card, flux.Good, now)
fmt.Println(card.State) // Learning (yksi vaihe jäljellä)

// Toinen kertaus — siirtyy Review-tilaan
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.State) // Review
fmt.Println(card.Due)   // ~2 päivää tästä hetkestä

// Kolmas kertaus — aikaväli kasvaa jokaisen onnistuneen muistamisen jälkeen
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.Due)   // ~10 päivää tästä hetkestä

// Tarkista muistamistodennäköisyys minä tahansa ajanhetkenä
r := s.Retrievability(card, card.Due)
fmt.Printf("%.0f%%\n", r*100) // ~90% (vastaa DesiredRetention-arvoa)

// Esikatsele kaikkia neljää arviointitulosta ennen vahvistamista
preview := s.PreviewCard(card, card.Due)
for _, rating := range []flux.Rating{flux.Again, flux.Hard, flux.Good, flux.Easy} {
    fmt.Printf("%s → %s\n", rating, preview[rating].Due)
}
```

Katso [`examples/`](examples/) täydellisistä ajettavista ohjelmista, jotka kattavat peruselinkaaren, parametrien optimoinnin ja kertauslokien uudelleenajoituksen.

## API-yleiskatsaus

### Perustyypit

```go
// Card sisältää yksittäisen muistikortin ajoitustilan.
type Card struct {
    CardID     int64
    State      State      // Learning, Review tai Relearning
    Step       *int       // nykyinen oppimis-/uudelleenoppimisvaihe (nil Review-tilassa)
    Stability  *float64   // muistin vakaus päivinä (nil ennen ensimmäistä kertausta)
    Difficulty *float64   // kohteen vaikeus (nil ennen ensimmäistä kertausta)
    Due        time.Time
    LastReview *time.Time
}

// Rating edustaa käyttäjän muistamisarviota.
type Rating int // Again=1, Hard=2, Good=3, Easy=4

// State edustaa kortin oppimisvaihetta.
type State int // Learning=1, Review=2, Relearning=3

// ReviewLog tallentaa yksittäisen kertaustapahtuman.
type ReviewLog struct {
    CardID         int64
    Rating         Rating
    ReviewDatetime time.Time
    ReviewDuration *int // millisekuntia, valinnainen
}
```

### Scheduler

```go
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error)
```

| Metodi | Kuvaus |
|--------|--------|
| `ReviewCard(card Card, rating Rating, now time.Time) (Card, ReviewLog)` | Käsittele kertaus ja palauta päivitetty kortti ja loki |
| `PreviewCard(card Card, now time.Time) map[Rating]Card` | Esikatsele kaikkien neljän arvioinnin tulokset |
| `RescheduleCard(card Card, logs []ReviewLog) (Card, error)` | Toista kertauslokit kortin tilan uudelleenrakentamiseksi |
| `Retrievability(card Card, now time.Time) float64` | Laske muistamistodennäköisyys annettuna ajanhetkenä |

### SchedulerConfig

```go
type SchedulerConfig struct {
    Parameters       [21]float64     // nolla-arvo -> DefaultParameters
    DesiredRetention float64         // nolla-arvo -> 0.9
    LearningSteps    []time.Duration // nil -> [1m, 10m]
    RelearningSteps  []time.Duration // nil -> [10m]
    MaximumInterval  int             // nolla-arvo -> 36500 päivää
    DisableFuzzing   bool            // nolla-arvo -> false (hajautus käytössä)
}
```

### Parameters

```go
var DefaultParameters [21]float64 // FSRS v6 oletusparametrit py-fsrs:stä
var LowerBounds [21]float64
var UpperBounds [21]float64

func ValidateParameters(p [21]float64) error
```

## Optimoija

`optimizer`-alipaketti kouluttaa FSRS-parametreja todellisesta kertaushistoriasta ja laskee optimaaliset muistamisprosenttitavoitteet.

```go
import "github.com/sky-flux/flux/optimizer"

// Kerää kertauslokit sovelluksestasi (esim. tietokannasta).
// Jokainen loki tallentaa mikä kortti kertattiin, arviointi ja ajankohta.
logs := []flux.ReviewLog{
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day1},
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day3},
    // ... satoja tai tuhansia todellisia kertauksia
}

opt := optimizer.NewOptimizer(optimizer.OptimizerConfig{})

// Kouluta henkilökohtaiset parametrit kertaushistoriasta
params, err := opt.ComputeOptimalParameters(ctx, logs)

// Käytä optimoituja parametreja uudessa ajoittajassa
s, _ := flux.NewScheduler(flux.SchedulerConfig{Parameters: params})

// Valinnainen: etsi muistamisprosenttitavoite, joka minimoi kokonaiskertauskustannukset.
// Edellyttää ReviewDuration-kentän asettamista jokaisessa lokissa.
retention, err := opt.ComputeOptimalRetention(ctx, params, logs)
```

### OptimizerConfig

| Kenttä | Oletus | Kuvaus |
|--------|--------|--------|
| `Epochs` | 5 | Koulutuskierrokset |
| `MiniBatchSize` | 512 | Kertaukset per minierä |
| `LearningRate` | 0.04 | Adam-alkuoppimisaste |
| `MaxSeqLen` | 64 | Maks. kertaukset per kortti |

## Suorituskyky

Ympäristö: Mac Mini (Apple M4 Pro, 64 GB RAM, 2T SSD), macOS 26.2, Go 1.26 darwin/arm64

| Vertailukohde | Tulos | Tavoite |
|---------------|-------|---------|
| ReviewCard | 181 ns/op, 80 B, 6 allocs | < 500 ns |
| GetRetrievability | 24 ns/op, 0 B, 0 allocs | < 100 ns |
| PreviewCard | 820 ns/op, 1112 B, 31 allocs | < 2 us |
| Optimize1000 | 0.49 s | < 2 s |
| Optimize10000 | 4.59 s | < 15 s |

## Yhdenmukaisuus py-fsrs:n kanssa

flux on rivi riviltä portattu viitetoteutuksesta [py-fsrs](https://github.com/open-spaced-repetition/py-fsrs). Kaikki 21 FSRS v6 -parametria, muistitilayhtälöt, vakaus-/vaikeuspäivityskaavat ja aikavälin laskentalogiikka vastaavat Python-viitetoteutusta. Testisarja validoi tulosteiden yhdenmukaisuuden py-fsrs:n kanssa samoilla syötteillä ja parametrisarjoilla.

## Esimerkit

[`examples/`](examples/)-hakemisto sisältää täydellisiä ajettavia ohjelmia:

| Esimerkki | Kuvaus | Suorita |
|-----------|--------|---------|
| [`basic`](examples/basic/) | Kortin luonti, kertaussilmukka, esikatselu | `go run ./examples/basic/` |
| [`optimizer`](examples/optimizer/) | Parametrien koulutus, optimaalinen muistamisprosentti | `go run ./examples/optimizer/` |
| [`reschedule`](examples/reschedule/) | Kertauslokien toisto tilan uudelleenrakentamiseksi | `go run ./examples/reschedule/` |

## Osallistuminen

Katso [CONTRIBUTING.md](CONTRIBUTING.md) ohjeet projektin kehittämiseen osallistumisesta.

## Lisenssi

[MIT](LICENSE)
