# flux

[English](README.md) | [简体中文](README_zh-CN.md) | [繁體中文](README_zh-TW.md) | [Deutsch](README_de.md) | [Suomi](README_fi.md) | [Español](README_es.md)

[![CI](https://img.shields.io/github/actions/workflow/status/sky-flux/flux/ci.yml?branch=main&label=CI)](https://github.com/sky-flux/flux/actions)
[![codecov](https://codecov.io/github/sky-flux/flux/graph/badge.svg?token=YT941R23LJ)](https://codecov.io/github/sky-flux/flux)
[![Go Report Card](https://goreportcard.com/badge/github.com/sky-flux/flux)](https://goreportcard.com/report/github.com/sky-flux/flux)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**FSRS v6** 间隔重复算法的纯 Go 实现。无标准库以外的依赖。

flux 提供了完整的闪卡调度引擎：复习调度、记忆状态追踪、基于复习历史的参数优化，以及通过蒙特卡洛模拟计算最优保留率。

## 特性

- **FSRS v6 算法** -- 完整实现最新的自由间隔重复调度器，包含 21 个可训练参数
- **零依赖** -- 仅使用 Go 标准库
- **卡片生命周期管理** -- 学习中、复习中、重新学习中状态，支持可配置的步骤时长
- **参数优化器** -- 使用 Adam 优化器和余弦退火的小批量梯度下降，从复习日志中训练参数
- **最优保留率** -- 蒙特卡洛模拟，寻找使总复习成本最小化的保留率目标
- **可提取性** -- 计算任意时间点任意卡片的回忆概率
- **预览与重新调度** -- 在提交前预览所有评分结果，或重放复习日志以重建卡片状态
- **间隔模糊化** -- 可选的随机化处理，分散复习时间以避免聚集
- **JSON 序列化** -- Card、Rating、State、Scheduler 和 ReviewLog 均实现了 JSON 编解码
- **确定性与可测试性** -- 可禁用模糊化以获得可复现的测试结果

## 快速开始

```bash
go get github.com/sky-flux/flux
```

创建一张卡片，进行多次复习，观察调度如何自适应：

```go
s, _ := flux.NewScheduler(flux.SchedulerConfig{DesiredRetention: 0.9})
card := flux.NewCard(1)
now := time.Now()

// 第一次复习 — 卡片在学习步骤中移动
card, _ = s.ReviewCard(card, flux.Good, now)
fmt.Println(card.State) // Learning（还有一步）

// 第二次复习 — 毕业进入复习阶段
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.State) // Review
fmt.Println(card.Due)   // 约 2 天后

// 第三次复习 — 每次成功回忆后间隔增长
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.Due)   // 约 10 天后

// 在任意时间点检查回忆概率
r := s.Retrievability(card, card.Due)
fmt.Printf("%.0f%%\n", r*100) // ~90%（与 DesiredRetention 匹配）

// 在提交前预览所有四种评分结果
preview := s.PreviewCard(card, card.Due)
for _, rating := range []flux.Rating{flux.Again, flux.Hard, flux.Good, flux.Easy} {
    fmt.Printf("%s → %s\n", rating, preview[rating].Due)
}
```

完整的可运行程序请参见 [`examples/`](examples/)，涵盖基本生命周期、参数优化和复习日志重新调度。

## API 概览

### 核心类型

```go
// Card 保存单张闪卡的调度状态。
type Card struct {
    CardID     int64
    State      State      // Learning、Review 或 Relearning
    Step       *int       // 当前学习/重新学习步骤（Review 状态下为 nil）
    Stability  *float64   // 记忆稳定性，单位为天（首次复习前为 nil）
    Difficulty *float64   // 项目难度（首次复习前为 nil）
    Due        time.Time
    LastReview *time.Time
}

// Rating 表示用户的回忆评估。
type Rating int // Again=1, Hard=2, Good=3, Easy=4

// State 表示卡片的学习阶段。
type State int // Learning=1, Review=2, Relearning=3

// ReviewLog 记录一次复习事件。
type ReviewLog struct {
    CardID         int64
    Rating         Rating
    ReviewDatetime time.Time
    ReviewDuration *int // 毫秒，可选
}
```

### Scheduler

```go
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error)
```

| 方法 | 描述 |
|------|------|
| `ReviewCard(card Card, rating Rating, now time.Time) (Card, ReviewLog)` | 处理一次复习并返回更新后的卡片和日志 |
| `PreviewCard(card Card, now time.Time) map[Rating]Card` | 预览所有四种评分的结果 |
| `RescheduleCard(card Card, logs []ReviewLog) (Card, error)` | 重放复习日志以重建卡片状态 |
| `Retrievability(card Card, now time.Time) float64` | 计算给定时间的回忆概率 |

### SchedulerConfig

```go
type SchedulerConfig struct {
    Parameters       [21]float64     // 零值 -> DefaultParameters
    DesiredRetention float64         // 零值 -> 0.9
    LearningSteps    []time.Duration // nil -> [1m, 10m]
    RelearningSteps  []time.Duration // nil -> [10m]
    MaximumInterval  int             // 零值 -> 36500 天
    DisableFuzzing   bool            // 零值 -> false（启用模糊化）
}
```

### Parameters

```go
var DefaultParameters [21]float64 // FSRS v6 默认参数，来自 py-fsrs
var LowerBounds [21]float64
var UpperBounds [21]float64

func ValidateParameters(p [21]float64) error
```

## 优化器

`optimizer` 子包从真实复习历史中训练 FSRS 参数，并计算最优保留率目标。

```go
import "github.com/sky-flux/flux/optimizer"

// 从你的应用（如数据库）收集复习日志。
// 每条日志记录了被复习的卡片、评分和时间。
logs := []flux.ReviewLog{
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day1},
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day3},
    // ... 数百或数千条真实复习记录
}

opt := optimizer.NewOptimizer(optimizer.OptimizerConfig{})

// 从复习历史训练个性化参数
params, err := opt.ComputeOptimalParameters(ctx, logs)

// 在新的调度器中使用优化后的参数
s, _ := flux.NewScheduler(flux.SchedulerConfig{Parameters: params})

// 可选：寻找使总复习成本最小化的保留率目标。
// 需要在每条日志中设置 ReviewDuration。
retention, err := opt.ComputeOptimalRetention(ctx, params, logs)
```

### OptimizerConfig

| 字段 | 默认值 | 描述 |
|------|--------|------|
| `Epochs` | 5 | 训练轮数 |
| `MiniBatchSize` | 512 | 每个小批量的复习数 |
| `LearningRate` | 0.04 | Adam 初始学习率 |
| `MaxSeqLen` | 64 | 每张卡片的最大复习数 |

## 性能

环境：Mac Mini（Apple M4 Pro，64 GB RAM，2T SSD），macOS 26.2，Go 1.26 darwin/arm64

| 基准测试 | 结果 | 目标 |
|----------|------|------|
| ReviewCard | 181 ns/op, 80 B, 6 allocs | < 500 ns |
| GetRetrievability | 24 ns/op, 0 B, 0 allocs | < 100 ns |
| PreviewCard | 820 ns/op, 1112 B, 31 allocs | < 2 us |
| Optimize1000 | 0.49 s | < 2 s |
| Optimize10000 | 4.59 s | < 15 s |

## 与 py-fsrs 的一致性

flux 是参考实现 [py-fsrs](https://github.com/open-spaced-repetition/py-fsrs) 的逐行移植。所有 21 个 FSRS v6 参数、记忆状态方程、稳定性/难度更新公式以及间隔计算逻辑均与 Python 参考实现一致。测试套件验证了在相同输入和参数集下与 py-fsrs 的输出一致性。

## 示例

[`examples/`](examples/) 目录包含完整的可运行程序：

| 示例 | 描述 | 运行 |
|------|------|------|
| [`basic`](examples/basic/) | 卡片创建、复习循环、预览 | `go run ./examples/basic/` |
| [`optimizer`](examples/optimizer/) | 参数训练、最优保留率 | `go run ./examples/optimizer/` |
| [`reschedule`](examples/reschedule/) | 重放复习日志以重建状态 | `go run ./examples/reschedule/` |

## 贡献

参见 [CONTRIBUTING.md](CONTRIBUTING.md) 了解如何为本项目做出贡献。

## 许可证

[MIT](LICENSE)
