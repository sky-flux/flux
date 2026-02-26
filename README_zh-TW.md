# flux

[English](README.md) | [简体中文](README_zh-CN.md) | [繁體中文](README_zh-TW.md) | [Deutsch](README_de.md) | [Suomi](README_fi.md) | [Español](README_es.md)

[![CI](https://img.shields.io/github/actions/workflow/status/sky-flux/flux/ci.yml?branch=main&label=CI)](https://github.com/sky-flux/flux/actions)
[![codecov](https://codecov.io/github/sky-flux/flux/graph/badge.svg?token=YT941R23LJ)](https://codecov.io/github/sky-flux/flux)
[![Go Report Card](https://goreportcard.com/badge/github.com/sky-flux/flux)](https://goreportcard.com/report/github.com/sky-flux/flux)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**FSRS v6** 間隔重複演算法的純 Go 實現。無標準函式庫以外的依賴。

flux 提供了完整的閃卡排程引擎：複習排程、記憶狀態追蹤、基於複習歷史的參數最佳化，以及透過蒙地卡羅模擬計算最佳保留率。

## 特性

- **FSRS v6 演算法** -- 完整實現最新的自由間隔重複排程器，包含 21 個可訓練參數
- **零依賴** -- 僅使用 Go 標準函式庫
- **卡片生命週期管理** -- 學習中、複習中、重新學習中狀態，支援可設定的步驟時長
- **參數最佳化器** -- 使用 Adam 最佳化器和餘弦退火的小批次梯度下降，從複習日誌中訓練參數
- **最佳保留率** -- 蒙地卡羅模擬，尋找使總複習成本最小化的保留率目標
- **可提取性** -- 計算任意時間點任意卡片的回憶機率
- **預覽與重新排程** -- 在提交前預覽所有評分結果，或重播複習日誌以重建卡片狀態
- **間隔模糊化** -- 可選的隨機化處理，分散複習時間以避免聚集
- **JSON 序列化** -- Card、Rating、State、Scheduler 和 ReviewLog 均實現了 JSON 編解碼
- **確定性與可測試性** -- 可停用模糊化以獲得可複現的測試結果

## 快速開始

```bash
go get github.com/sky-flux/flux
```

建立一張卡片，進行多次複習，觀察排程如何自適應：

```go
s, _ := flux.NewScheduler(flux.SchedulerConfig{DesiredRetention: 0.9})
card := flux.NewCard(1)
now := time.Now()

// 第一次複習 — 卡片在學習步驟中移動
card, _ = s.ReviewCard(card, flux.Good, now)
fmt.Println(card.State) // Learning（還有一步）

// 第二次複習 — 畢業進入複習階段
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.State) // Review
fmt.Println(card.Due)   // 約 2 天後

// 第三次複習 — 每次成功回憶後間隔增長
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.Due)   // 約 10 天後

// 在任意時間點檢查回憶機率
r := s.Retrievability(card, card.Due)
fmt.Printf("%.0f%%\n", r*100) // ~90%（與 DesiredRetention 匹配）

// 在提交前預覽所有四種評分結果
preview := s.PreviewCard(card, card.Due)
for _, rating := range []flux.Rating{flux.Again, flux.Hard, flux.Good, flux.Easy} {
    fmt.Printf("%s → %s\n", rating, preview[rating].Due)
}
```

完整的可執行程式請參見 [`examples/`](examples/)，涵蓋基本生命週期、參數最佳化和複習日誌重新排程。

## API 概覽

### 核心型別

```go
// Card 保存單張閃卡的排程狀態。
type Card struct {
    CardID     int64
    State      State      // Learning、Review 或 Relearning
    Step       *int       // 目前學習/重新學習步驟（Review 狀態下為 nil）
    Stability  *float64   // 記憶穩定性，單位為天（首次複習前為 nil）
    Difficulty *float64   // 項目難度（首次複習前為 nil）
    Due        time.Time
    LastReview *time.Time
}

// Rating 表示使用者的回憶評估。
type Rating int // Again=1, Hard=2, Good=3, Easy=4

// State 表示卡片的學習階段。
type State int // Learning=1, Review=2, Relearning=3

// ReviewLog 記錄一次複習事件。
type ReviewLog struct {
    CardID         int64
    Rating         Rating
    ReviewDatetime time.Time
    ReviewDuration *int // 毫秒，可選
}
```

### Scheduler

```go
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error)
```

| 方法 | 描述 |
|------|------|
| `ReviewCard(card Card, rating Rating, now time.Time) (Card, ReviewLog)` | 處理一次複習並回傳更新後的卡片和日誌 |
| `PreviewCard(card Card, now time.Time) map[Rating]Card` | 預覽所有四種評分的結果 |
| `RescheduleCard(card Card, logs []ReviewLog) (Card, error)` | 重播複習日誌以重建卡片狀態 |
| `Retrievability(card Card, now time.Time) float64` | 計算給定時間的回憶機率 |

### SchedulerConfig

```go
type SchedulerConfig struct {
    Parameters       [21]float64     // 零值 -> DefaultParameters
    DesiredRetention float64         // 零值 -> 0.9
    LearningSteps    []time.Duration // nil -> [1m, 10m]
    RelearningSteps  []time.Duration // nil -> [10m]
    MaximumInterval  int             // 零值 -> 36500 天
    DisableFuzzing   bool            // 零值 -> false（啟用模糊化）
}
```

### Parameters

```go
var DefaultParameters [21]float64 // FSRS v6 預設參數，來自 py-fsrs
var LowerBounds [21]float64
var UpperBounds [21]float64

func ValidateParameters(p [21]float64) error
```

## 最佳化器

`optimizer` 子套件從真實複習歷史中訓練 FSRS 參數，並計算最佳保留率目標。

```go
import "github.com/sky-flux/flux/optimizer"

// 從你的應用程式（如資料庫）收集複習日誌。
// 每筆日誌記錄了被複習的卡片、評分和時間。
logs := []flux.ReviewLog{
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day1},
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day3},
    // ... 數百或數千筆真實複習記錄
}

opt := optimizer.NewOptimizer(optimizer.OptimizerConfig{})

// 從複習歷史訓練個人化參數
params, err := opt.ComputeOptimalParameters(ctx, logs)

// 在新的排程器中使用最佳化後的參數
s, _ := flux.NewScheduler(flux.SchedulerConfig{Parameters: params})

// 可選：尋找使總複習成本最小化的保留率目標。
// 需要在每筆日誌中設定 ReviewDuration。
retention, err := opt.ComputeOptimalRetention(ctx, params, logs)
```

### OptimizerConfig

| 欄位 | 預設值 | 描述 |
|------|--------|------|
| `Epochs` | 5 | 訓練輪數 |
| `MiniBatchSize` | 512 | 每個小批次的複習數 |
| `LearningRate` | 0.04 | Adam 初始學習率 |
| `MaxSeqLen` | 64 | 每張卡片的最大複習數 |

## 效能

環境：Mac Mini（Apple M4 Pro，64 GB RAM，2T SSD），macOS 26.2，Go 1.26 darwin/arm64

| 基準測試 | 結果 | 目標 |
|----------|------|------|
| ReviewCard | 181 ns/op, 80 B, 6 allocs | < 500 ns |
| GetRetrievability | 24 ns/op, 0 B, 0 allocs | < 100 ns |
| PreviewCard | 820 ns/op, 1112 B, 31 allocs | < 2 us |
| Optimize1000 | 0.49 s | < 2 s |
| Optimize10000 | 4.59 s | < 15 s |

## 與 py-fsrs 的一致性

flux 是參考實現 [py-fsrs](https://github.com/open-spaced-repetition/py-fsrs) 的逐行移植。所有 21 個 FSRS v6 參數、記憶狀態方程式、穩定性/難度更新公式以及間隔計算邏輯均與 Python 參考實現一致。測試套件驗證了在相同輸入和參數集下與 py-fsrs 的輸出一致性。

## 範例

[`examples/`](examples/) 目錄包含完整的可執行程式：

| 範例 | 描述 | 執行 |
|------|------|------|
| [`basic`](examples/basic/) | 卡片建立、複習迴圈、預覽 | `go run ./examples/basic/` |
| [`optimizer`](examples/optimizer/) | 參數訓練、最佳保留率 | `go run ./examples/optimizer/` |
| [`reschedule`](examples/reschedule/) | 重播複習日誌以重建狀態 | `go run ./examples/reschedule/` |

## 貢獻

參見 [CONTRIBUTING.md](CONTRIBUTING.md) 了解如何為本專案做出貢獻。

## 授權條款

[MIT](LICENSE)
