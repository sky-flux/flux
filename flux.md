# flux

> Go FSRS v6 Scheduler + Optimizer
>
> `github.com/sky-flux/flux` · MIT License

---

## 目录

1. [定位](#定位)
2. [用户画像与贡献路径](#用户画像与贡献路径)
3. [FSRS v6 算法规格](#fsrs-v6-算法规格)
4. [Optimizer 优化机制](#optimizer-优化机制)
5. [Scheduler 状态机](#scheduler-状态机)
6. [公共 API](#公共-api)
7. [TDD 策略与 100% 覆盖计划](#tdd-策略与-100-覆盖计划)
8. [迭代计划 v0.1.0 → v1.0.0](#迭代计划-v010--v100)
9. [项目文件清单](#项目文件清单)
10. [v5 → v6 差异清单](#v5--v6-差异清单)
11. [参考文献](#参考文献)

---

## 定位

flux 是 FSRS v6 算法的纯 Go 实现，包含 Scheduler 和 Optimizer 两个核心能力。

截至 2025 年 2 月，Go 生态中仅有 go-fsrs（v5 Scheduler，无 Optimizer）。flux 补全这两个空白：**v6 Scheduler + v6 Optimizer**。

| 参考实现 | 语言 | Scheduler | Optimizer | 角色 |
|---------|------|-----------|-----------|------|
| py-fsrs | Python | v6 ✅ | v6 ✅ (PyTorch) | **算法对标** |
| fsrs-rs | Rust | v6 ✅ | v6 ✅ (burn-rs) | Optimizer 参考 |
| go-fsrs v3 | Go | v5 ✅ | ❌ | Go 生态现状参考 |
| riff (SiYuan) | Go | v5 ✅ (via go-fsrs) | ❌ | Go 集成案例参考 |
| fsrs4anki Wiki | - | 全版本 | 全版本 | 公式 + 优化机制权威来源 |
| anki-revlogs-10k | - | - | - | Optimizer 验证数据集 |

### 设计原则

1. **算法精确**：逐公式对齐 py-fsrs v6 + fsrs4anki Wiki FSRS-6，精度 1e-4
2. **零依赖 Scheduler**：仅依赖 Go 标准库（math, time, errors, encoding/json）
3. **隔离 Optimizer**：子包 `flux/optimizer`，仅依赖标准库（无需 gonum）
4. **TDD 驱动**：测试先行，v1.0.0 发布时 100% 单元测试覆盖率
5. **原生 API**：flux 只暴露自己的 API，不兼容 go-fsrs 的接口
6. **Go 惯用风格**：实现标准接口（fmt.Stringer、json.Marshaler、encoding.TextMarshaler）；零值可用的配置结构体；包级哨兵错误（errors.Is）；方法命名不加 Get 前缀

### 为什么不提供 go-fsrs 兼容层

flux 的 Card 结构（指针字段、py-fsrs 风格）与 go-fsrs 的 Card 结构（零值、含 Reps/Lapses/ElapsedDays 等）类型不兼容。强行做兼容层会引入大量类型转换代码和持续维护负担，且 Repeat 模式（一次返回四种评分结果）有 3/4 计算浪费。如果存量项目（如 SiYuan riff）需要迁移，可以在自己的代码中写薄适配层。

---

## 用户画像与贡献路径

基于 Mozilla Personas & Pathways 方法论。

### Persona 1：Go SRS 应用开发者

**Mei**，后端工程师，正在用 Go 开发词汇学习 App。当前通过 HTTP 调用 Python py-fsrs 服务做间隔重复调度，想消除跨语言依赖。

| 阶段 | 行为 |
|------|------|
| Discovery | 在 awesome-fsrs 或 Go 包搜索中发现 flux |
| First Contact | 阅读 README，`go get github.com/sky-flux/flux`，跑通 examples/basic |
| Participation | 在 Issues 中报告边界 case 的 bug |
| Sustained | 集成到生产系统，贡献 benchmark 对比数据 |
| Leadership | 贡献新功能或帮助维护 |

**需要到位**：清晰的 Quick Start、godoc 注释、`go get` 即用、对齐测试证明正确性。

### Persona 2：FSRS 算法研究者

**Liam**，研究记忆模型的硕士生，想在 Go 环境中复现或扩展 FSRS 优化实验。

| 阶段 | 行为 |
|------|------|
| Discovery | 从 fsrs4anki Wiki 的 Awesome FSRS 列表找到 flux |
| First Contact | 阅读 flux.md 中的算法规格，对比 Wiki 公式 |
| Participation | 用 anki-revlogs-10k 数据集跑 Optimizer，对比 py-fsrs 结果 |
| Sustained | 提交 Optimizer 精度改进的 PR |
| Leadership | 贡献新版本算法（如 FSRS-7）的迁移 |

**需要到位**：flux.md 中完整的公式推导、Optimizer 数学原理、可复现测试用例。

### Persona 3：SiYuan / Anki 插件开发者

**Koji**，SiYuan 笔记用户和插件开发者，想将 riff 从 go-fsrs v5 升级到 v6。

| 阶段 | 行为 |
|------|------|
| Discovery | 在 go-fsrs Issues 中看到 flux 的链接 |
| First Contact | 阅读 README，对比 API 差异 |
| Participation | 在 SiYuan riff 中编写适配层，引入 flux |
| Sustained | 反馈迁移过程中发现的问题 |
| Leadership | 成为 flux 在 SiYuan 生态的推广者 |

**需要到位**：README 中 API 速览、flux.md 中清晰的类型定义、迁移说明（作为 Wiki 或 FAQ）。

---

## FSRS v6 算法规格

### 符号

| 符号 | 含义 |
|------|------|
| R | 可提取性 Retrievability（回忆概率） |
| S | 稳定性 Stability（R = 90% 时的间隔天数） |
| D | 难度 Difficulty，D ∈ [1, 10] |
| G | 评分 Grade：Again = 1, Hard = 2, Good = 3, Easy = 4 |
| w[i] | 第 i 个参数（共 21 个，i ∈ [0, 20]） |

### 21 个参数

```
默认值（py-fsrs v6 / fsrs4anki Wiki FSRS-6）：
w = [0.212, 1.2931, 2.3065, 8.2956,
     6.4133, 0.8334, 3.0194, 0.001,
     1.8722, 0.1666, 0.796, 1.4835,
     0.0614, 0.2629, 1.6483, 0.6014,
     1.8729, 0.5425, 0.0912, 0.0658,
     0.1542]

用途：
w[0..3]  初始稳定性 S₀(G)，G = 1..4 各对应一个
w[4]     初始难度 D₀(1)（首次评 Again 时的基准）
w[5]     初始难度衰减系数
w[6]     难度更新步长
w[7]     均值回归强度
w[8]     成功复习稳定性增长基础
w[9]     成功复习中 S 的衰减指数
w[10]    成功复习中 R 的加速系数
w[11]    遗忘后稳定性基础
w[12]    遗忘后 D 衰减指数
w[13]    遗忘后 S 增长指数
w[14]    遗忘后 R 加速系数
w[15]    Hard 评分惩罚乘子
w[16]    Easy 评分奖励乘子
w[17]    同日复习稳定性增长率
w[18]    同日复习评分偏移
w[19]    同日复习 S 衰减指数      ← v6 新增
w[20]    遗忘曲线衰减指数（可训练）  ← v6 新增
```

### 参数边界

```
下界：[0.001, 0.001, 0.001, 0.001,
       1.0, 0.001, 0.001, 0.001,
       0.0, 0.0, 0.001, 0.001,
       0.001, 0.001, 0.0, 0.0,
       1.0, 0.0, 0.0, 0.0,
       0.1]

上界：[100.0, 100.0, 100.0, 100.0,
       10.0, 4.0, 4.0, 0.75,
       4.5, 0.8, 3.5, 5.0,
       0.25, 0.9, 4.0, 1.0,
       6.0, 2.0, 2.0, 0.8,
       0.8]
```

### 预计算常量

```
DECAY   = -w[20]
FACTOR  = 0.9 ^ (1 / DECAY) - 1
```

FACTOR 保证 R(S, S) = 90%。v5 中 DECAY 固定为 -0.5、FACTOR 固定为 19/81。

### Clamp 函数

```
clamp_s(s) = max(s, 0.001)
clamp_d(d) = clamp(d, 1, 10)
```

### 遗忘曲线

```
R(t, S) = (1 + FACTOR · t / S) ^ DECAY
```

**v6 变化**：DECAY = -w[20] 可训练。v5 中固定为 -0.5。

### 初始稳定性

```
S₀(G) = clamp_s(w[G - 1])
```

### 初始难度

```
D₀(G) = w[4] - e^(w[5] · (G - 1)) + 1
```

首次复习时 clamp 到 [1, 10]。在 nextDifficulty 的均值回归目标计算中**不** clamp。

### 下次间隔

```
I(r, S) = round((S / FACTOR) · (r ^ (1 / DECAY) - 1))

clamp 到 [1, maximum_interval]
```

其中 r = desired_retention。

### 同日复习稳定性（short-term）

```
SInc = e^(w[17] · (G - 3 + w[18])) · S^(-w[19])

若 G ∈ {Good, Easy}：SInc = max(SInc, 1.0)

S' = clamp_s(S · SInc)
```

**v6 变化**：新增 S^(-w[19]) 项。小 S 增长快、大 S 增长慢，S 最终收敛到 SInc = 1 的平衡点。

### 难度更新

```
ΔD = -w[6] · (G - 3)
D'  = D + (10 - D) · ΔD / 9              ← 线性阻尼
D'' = w[7] · D₀(Easy) + (1 - w[7]) · D'  ← 均值回归
D'' = clamp_d(D'')
```

均值回归目标：D₀(Easy) 即 D₀(4)，取**不 clamp** 的值。

### 成功复习后的新稳定性（recall）

用于 Hard / Good / Easy：

```
hardPenalty = w[15] if G = Hard, else 1
easyBonus   = w[16] if G = Easy, else 1

S'_r(D, S, R, G) = S · (1 + e^w[8]
                            · (11 - D)
                            · S^(-w[9])
                            · (e^((1 - R) · w[10]) - 1)
                            · hardPenalty
                            · easyBonus)
```

### 遗忘后的新稳定性（forget）

用于 Again：

```
长期公式：
S'_f_long = w[11] · D^(-w[12]) · ((S + 1)^w[13] - 1) · e^((1 - R) · w[14])

短期上限：
S'_f_short = S / e^(w[17] · w[18])

S'_f = min(S'_f_long, S'_f_short)
```

**v6 变化**：取 min(long, short)。短期上限确保遗忘后稳定性不超过「同日 Again 的回退值」。

### Fuzz

当 DisableFuzzing = false（默认）且卡片最终进入 Review 状态时，对间隔天数做模糊化以防止复习堆积。

```
若 interval < 2.5 天：不 fuzz

Fuzz 范围表：
  [2.5,  7.0)  → factor = 0.15
  [7.0,  20.0) → factor = 0.10
  [20.0, +∞)   → factor = 0.05

delta = 1.0 + Σ(factor · max(min(interval, end) - start, 0))
min_ivl = max(2, round(interval - delta))
max_ivl = min(round(interval + delta), maximum_interval)
min_ivl = min(min_ivl, max_ivl)

fuzzed = min(round(rand() · (max_ivl - min_ivl + 1) + min_ivl), maximum_interval)
```

Learning / Relearning 的步骤间隔不 fuzz。

---

## Optimizer 优化机制

基于 fsrs4anki Wiki「The mechanism of optimization」。

### 理论基础

FSRS 基于 DSR（Difficulty, Stability, Retrievability）记忆模型。Optimizer 的目标是从用户的时序复习日志中学习记忆规律，估计最优的 21 个参数。

核心方法：**最大似然估计（MLE）+ 时序反向传播（BPTT）**。

### 损失函数推导

设遗忘曲线函数为 R(t, S)。在稳定性 S 下经过 t 天后：

```
回忆成功的概率：P(r=1, t | S) = R(t, S)
遗忘的概率：    P(r=0, t | S) = 1 - R(t, S)
```

根据 MLE，对数似然为：

```
log L = r · ln R(t, S) + (1 - r) · ln(1 - R(t, S))
```

取负号得损失函数（Binary Cross-Entropy）：

```
loss = -[r · ln R(t, S) + (1 - r) · ln(1 - R(t, S))]
```

其中 r = 0 if rating == Again, else 1。

将 S 替换为 DSR 模型的输出 DSR(r⃗, t⃗)，即可端到端训练：

```
R_pred = R(t, DSR(r⃗, t⃗ | θ))

loss 对 θ（21 个参数）求梯度 → 更新参数
```

**仅跨日复习（days_since_last_review ≥ 1）计入损失**。同日复习不影响遗忘曲线评估。

### Go 实现方案

py-fsrs 使用 PyTorch 自动微分。flux 使用数值微分替代：

**梯度计算**：双侧数值微分

```
∂L/∂w[i] ≈ (L(w[i] + ε) - L(w[i] - ε)) / (2ε)

ε = 1e-5
```

每轮对 21 个参数各算一次，共 42 次前向传播。

**优化器**：Adam

```
m[i] = β1 · m[i] + (1 - β1) · g[i]
v[i] = β2 · v[i] + (1 - β2) · g[i]²
m̂[i] = m[i] / (1 - β1^t)
v̂[i] = v[i] / (1 - β2^t)
w[i] = w[i] - lr · m̂[i] / (√v̂[i] + ε)

β1 = 0.9,  β2 = 0.999,  ε = 1e-8
```

**学习率调度**：Cosine Annealing

```
T_max = ceil(num_reviews / mini_batch_size) × epochs
lr_t  = 0.5 · lr · (1 + cos(π · t / T_max))
```

**参数约束**：每次 Adam 更新后 clamp 到 [LowerBounds, UpperBounds]。

### 训练流程

```
1. 预处理日志
   按 card_id 分组，组内按 review_datetime 排序。

2. 统计 num_reviews（跨日复习总数）
   若 num_reviews < mini_batch_size → 返回 DefaultParameters

3. 初始化
   params = DefaultParameters, Adam (lr=0.04), Cosine Annealing

4. 对每个 epoch (默认 5):
   a. 随机打乱 card_id 顺序（种子 42）
   b. 用当前 params 创建 Scheduler
   c. 对每张卡（截取前 max_seq_len=64 条）：
      - Card → ReviewCard(rating, datetime) → 新 Card
      - 若跨日复习 → 计算 step_loss (BCE)
      - 累计 512 条后 → 数值梯度 → Adam 更新 → Clamp → 重建 Scheduler
   d. 计算 epoch batch_loss
   e. 若 batch_loss < best_loss → 记录 best_params

5. 返回 best_params
```

### 最优保留率（ComputeOptimalRetention）

蒙特卡洛模拟：

```
1. 校验：≥ 512 条日志，ReviewDuration 不为 nil
2. 从日志统计评分概率和平均复习时长
3. 对候选 [0.70, 0.75, 0.80, 0.85, 0.90, 0.95]：
   模拟 1000 张卡片 × 1 年
   simulation_cost = total_duration / (retention × num_cards)
4. 返回 cost 最小的 retention
```

### 验证数据集：anki-revlogs-10k

HuggingFace `open-spaced-repetition/anki-revlogs-10k`，10k 个 Anki 用户的复习日志。

```
revlogs: card_id, day_offset, rating, state, duration, elapsed_days, elapsed_seconds
cards:   card_id, note_id, deck_id
decks:   deck_id, parent_id, preset_id
```

用途：`testdata/anki_revlogs_sample.json`，集成测试对比 py-fsrs 优化结果。

---

## Scheduler 状态机

flux 采用 py-fsrs 的 **step-based** 模型。每次 ReviewCard 只处理当前评分并返回新卡片。

### 卡片状态

```
Learning   → 新卡片，正在初始学习
Review     → 进入长期复习周期
Relearning → 遗忘后重新学习
```

### 配置

```
learning_steps   默认 [1m, 10m]
relearning_steps 默认 [10m]
```

### Learning 状态

**稳定性和难度更新**：

```
if stability == nil（首次复习）:
    S = S₀(G),  D = clamp_d(D₀(G))

elif days_since_last_review < 1（同日）:
    S = shortTermStability(S, G)
    D = nextDifficulty(D, G)

else（跨日）:
    R = retrievability(elapsed_days, S)
    S = nextStability(D, S, R, G)
    D = nextDifficulty(D, G)
```

**间隔和状态转换**：

```
若 learning_steps 为空，
或 step ≥ len(learning_steps) 且 G ∈ {Hard, Good, Easy}:
    → Review，step = nil，interval = nextInterval(S)

否则按 G 分发：
    Again:
        step = 0, interval = learning_steps[0]

    Hard:
        step 不变
        step == 0 且 len == 1 → interval = learning_steps[0] × 1.5
        step == 0 且 len ≥ 2  → interval = (learning_steps[0] + learning_steps[1]) / 2
        其他                   → interval = learning_steps[step]

    Good:
        step + 1 == len（最后一步） → Review, interval = nextInterval(S)
        否则 → step++, interval = learning_steps[step]

    Easy:
        → Review, interval = nextInterval(S)
```

### Review 状态

```
同日 → shortTermStability
跨日 → nextStability（基于 retrievability）
难度总是更新：nextDifficulty

Again:
    relearning_steps 非空 → Relearning(step=0)
    否则 → nextInterval
Hard / Good / Easy → nextInterval
```

### Relearning 状态

与 Learning 对称，使用 relearning_steps。

### 收尾

```
若 !disable_fuzzing 且最终状态 == Review → fuzz(interval)
card.due = review_datetime + interval
card.last_review = review_datetime
```

---

## 公共 API

### 核心类型

```go
type Rating int // Again=1, Hard=2, Good=3, Easy=4
type State  int // Learning=1, Review=2, Relearning=3

// Rating 和 State 均实现：
//   fmt.Stringer           — String() 返回 "Again"/"Hard"/"Good"/"Easy"
//   encoding.TextMarshaler — MarshalText / UnmarshalText（文本格式）
//   json.Marshaler         — MarshalJSON / UnmarshalJSON（JSON 序列化为字符串）
//   Rating 额外提供 IsValid() bool

type Card struct {
    CardID     int64      `json:"card_id"`
    State      State      `json:"state"`
    Step       *int       `json:"step"`       // nil when State=Review
    Stability  *float64   `json:"stability"`  // nil before first review
    Difficulty *float64   `json:"difficulty"` // nil before first review
    Due        time.Time  `json:"due"`
    LastReview *time.Time `json:"last_review"` // nil before first review
}

type ReviewLog struct {
    CardID         int64     `json:"card_id"`
    Rating         Rating    `json:"rating"`
    ReviewDatetime time.Time `json:"review_datetime"`
    ReviewDuration *int      `json:"review_duration,omitempty"` // milliseconds, optional
}
```

Card 和 ReviewLog 直接使用 `json.Marshal` / `json.Unmarshal`，无需自定义序列化方法。
指针字段序列化为 JSON `null`，语义清晰（"尚未复习"而非零值日期）。

### 哨兵错误

```go
var (
    ErrInvalidRating     = errors.New("flux: invalid rating")
    ErrInvalidParameters = errors.New("flux: parameters out of bounds")
    ErrCardIDMismatch    = errors.New("flux: card ID mismatch in review log")
    ErrInsufficientData  = errors.New("flux: insufficient review data for optimization")
)
```

调用方使用 `errors.Is(err, flux.ErrInvalidParameters)` 做错误分支判断。

### Scheduler

```go
type SchedulerConfig struct {
    Parameters       [21]float64     `json:"parameters"`        // 零值 → DefaultParameters
    DesiredRetention float64         `json:"desired_retention"` // 零值 → 0.9
    LearningSteps    []time.Duration `json:"learning_steps"`    // nil → [1m, 10m]；空切片 → 无步骤
    RelearningSteps  []time.Duration `json:"relearning_steps"`  // nil → [10m]；空切片 → 无步骤
    MaximumInterval  int             `json:"maximum_interval"`  // 零值 → 36500
    DisableFuzzing   bool            `json:"disable_fuzzing"`   // 零值 false → 启用 fuzz
}
```

**零值即默认**：`SchedulerConfig{}` 是合理的默认配置，`NewScheduler` 自动将零值字段填充为默认值。
`DisableFuzzing` 取代 `EnableFuzzing`，使零值（false）对应"启用 fuzz"的默认行为。
`LearningSteps` / `RelearningSteps` 使用 nil 与空切片区分"用默认"与"显式无步骤"。

```go
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error)
func (*Scheduler) ReviewCard(card Card, rating Rating, now time.Time) (Card, ReviewLog)
func (*Scheduler) Retrievability(card Card, now time.Time) float64
func (*Scheduler) RescheduleCard(card Card, logs []ReviewLog) (Card, error)
func (*Scheduler) PreviewCard(card Card, now time.Time) map[Rating]Card
```

Scheduler 实现 `json.Marshaler` / `json.Unmarshaler`，
可直接用 `json.Marshal(s)` / `json.Unmarshal(data, &s)` 做配置持久化。

### Optimizer（子包 `flux/optimizer`）

```go
type OptimizerConfig struct {
    Epochs        int     `json:"epochs"`          // 零值 → 5
    MiniBatchSize int     `json:"mini_batch_size"` // 零值 → 512
    LearningRate  float64 `json:"learning_rate"`   // 零值 → 0.04
    MaxSeqLen     int     `json:"max_seq_len"`     // 零值 → 64
}

func NewOptimizer(cfg OptimizerConfig) *Optimizer
func (*Optimizer) ComputeOptimalParameters(ctx context.Context, logs []ReviewLog) ([21]float64, error)
func (*Optimizer) ComputeOptimalRetention(ctx context.Context, params [21]float64, logs []ReviewLog) (float64, error)
func (*Optimizer) ComputeBatchLoss(params [21]float64, logs []ReviewLog) float64
```

长耗时方法（ComputeOptimalParameters、ComputeOptimalRetention）接受 `context.Context`，
支持超时控制和取消。`ComputeBatchLoss` 为纯计算、耗时短，不需要 context。

---

## TDD 策略与 100% 覆盖计划

### 原则

1. **Red → Green → Refactor**：每个函数先写失败测试，再实现，再重构
2. **一个 .go 对应一个 _test.go**
3. **100% 行覆盖**：CI 中 `go test -cover` 低于 100% 则 fail
4. **对齐测试优先于逻辑测试**：先用 py-fsrs 数值确定"正确答案"，再写代码让测试通过

### 测试分层

| 层级 | 测试类型 | 文件 |
|------|---------|------|
| L1 | 类型枚举（Rating/State 的值、String()、IsValid()） | rating_test.go, state_test.go |
| L2 | Card 生命周期（NewCard、clone、JSON 序列化/反序列化） | card_test.go |
| L3 | 参数校验（DefaultParameters 长度、边界、ValidateParameters） | parameters_test.go |
| L4 | 公式对齐（逐个数学函数对齐 py-fsrs，精度 1e-4） | algorithm_test.go |
| L5 | Fuzz 逻辑（范围、边界值、不 fuzz 条件） | fuzz_test.go |
| L6 | 状态机全路径（Learning/Review/Relearning 各分支） | scheduler_test.go |
| L7 | 序列对齐（5 个完整复习序列逐步对比 py-fsrs） | scheduler_test.go |
| L8 | Scheduler 辅助（PreviewCard、RescheduleCard、JSON 往返） | scheduler_test.go |
| L9 | Optimizer 单元（dataset/loss/adam/retention 各模块） | optimizer/*_test.go |
| L10 | Optimizer 收敛（合成数据训练后 loss 下降、参数偏差 < 10%） | optimizer/optimizer_test.go |
| L11 | 集成测试（anki-revlogs-10k 真实数据，与 py-fsrs 对比） | optimizer/integration_test.go |
| L12 | 性能基准（ReviewCard < 500ns, Optimizer 吞吐量） | bench_test.go, optimizer/bench_test.go |

### 覆盖率 100% 实施要点

- **error 分支**：用非法输入触发（越界参数、空切片、CardID 不匹配）
- **clamp 边界**：用极值输入触发 min/max 分支
- **fuzz 随机性**：固定种子测试，验证 interval ∈ [min_ivl, max_ivl]
- **JSON 错误**：畸形 JSON 触发反序列化错误
- **Optimizer 提前返回**：num_reviews < MiniBatchSize → 返回 DefaultParameters

---

## 迭代计划 v0.1.0 → v1.0.0

每个版本都是一个可用的增量。任务标记为 `[ ]` 待做、`[x]` 完成。

---

### v0.1.0 — 类型 + 算法内核

> 目标：所有 FSRS v6 数学函数可独立测试，100% 覆盖。
> 还不能调度卡片，但所有公式已经对齐 py-fsrs。

**源码**

- [x] `doc.go` — 包级 godoc 注释
- [x] `rating.go` — Rating 枚举（Again=1..Easy=4），String()，IsValid()，MarshalJSON/UnmarshalJSON，MarshalText/UnmarshalText
- [x] `state.go` — State 枚举（Learning=1..Relearning=3），String()，MarshalJSON/UnmarshalJSON，MarshalText/UnmarshalText
- [x] `card.go` — Card 结构（含 JSON tags）、NewCard()、clone()、setStability/setDifficulty/setStep/clearStep
- [x] `review_log.go` — ReviewLog 结构（含 JSON tags）
- [x] `errors.go` — 包级哨兵错误（ErrInvalidRating、ErrInvalidParameters、ErrCardIDMismatch、ErrInsufficientData）
- [x] `parameters.go` — DefaultParameters [21]float64、LowerBounds、UpperBounds、ValidateParameters()
- [x] `algorithm.go` — 全部纯数学函数：
  - [x] newAlgo(p) → algo{w, decay, factor}
  - [x] retrievability(elapsedDays, stability) → float64
  - [x] initStability(rating) → float64
  - [x] initDifficulty(rating, clamp) → float64
  - [x] nextInterval(stability, desiredRetention, maxIvl) → int
  - [x] shortTermStability(stability, rating) → float64
  - [x] nextDifficulty(difficulty, rating) → float64
  - [x] nextStability(d, s, r, rating) → float64
  - [x] nextRecallStability(d, s, r, rating) → float64
  - [x] nextForgetStability(d, s, r) → float64
  - [x] clampS(s), clampD(d)

**测试（L1–L4）**

- [x] `rating_test.go` — 枚举值、String()、IsValid()、JSON/Text 往返序列化、非法值 Unmarshal → error（~12 用例）
- [x] `state_test.go` — 枚举值、String()、JSON/Text 往返序列化（~10 用例）
- [x] `card_test.go` — NewCard 初始状态、clone 深拷贝、指针独立性、JSON 往返（nil 字段 → null）（~14 用例）
- [x] `review_log_test.go` — 构造、字段访问、JSON 往返（omitempty 行为）（~6 用例）
- [x] `errors_test.go` — errors.Is 断言、错误消息前缀（~4 用例）
- [x] `parameters_test.go` — DefaultParameters 长度=21、ValidateParameters 正例+各越界（~8 用例）
- [x] `algorithm_test.go` — 每个数学函数 ≥ 4 用例 + 边界值（~40 用例）：
  - [x] initStability: 4 ratings → w[0..3]
  - [x] initDifficulty: 4 ratings clamp=true + 1 clamp=false
  - [x] retrievability: t=0→1.0, t=S→0.9, t>S, S=min
  - [x] shortTermStability: 4 ratings + SInc clamp 验证
  - [x] nextDifficulty: Again/Good/Easy + D=1/D=10 边界
  - [x] nextRecallStability: Hard(penalty)/Good/Easy(bonus) + R=0.9/R=0.5
  - [x] nextForgetStability: 正常 + min(long,short) 切换点
  - [x] nextInterval: 正常 + clamp 下限 + clamp 上限

**工程**

- [x] `go.mod` — module github.com/sky-flux/flux, go 1.26
- [x] `.gitignore`
- [x] `LICENSE` — MIT 全文
- [x] `go test ./... -cover` — 100% 覆盖率

**交付标准**：`go test ./...` 全绿，覆盖率 100%，`go vet` 通过。

---

### v0.2.0 — Scheduler 核心

> 目标：可以用 ReviewCard 调度卡片。Learning / Review / Relearning 全路径可用。

**源码**

- [x] `fuzz.go` — fuzzRanges、fuzzRange()、applyFuzz()
- [x] `scheduler.go` — 全部 Scheduler 逻辑：
  - [x] SchedulerConfig 结构 + 零值默认填充（DisableFuzzing、nil slice 语义）
  - [x] NewScheduler(cfg) → (*Scheduler, error)（参数校验）
  - [x] ReviewCard(card, rating, now) → (Card, ReviewLog)
    - [x] Learning 状态：首次/同日/跨日 × Again/Hard/Good/Easy
    - [x] Review 状态：同日/跨日 × Again/Hard/Good/Easy
    - [x] Relearning 状态：同日/跨日 × Again/Hard/Good/Easy
    - [x] Fuzz 应用（仅 Review 状态）
    - [x] Due + LastReview 更新
  - [x] Retrievability(card, now) → float64

**测试（L5–L6）**

- [x] `fuzz_test.go` — ~10 用例：
  - [x] interval < 2.5 → 不 fuzz
  - [x] interval = 3 → fuzz range [2.5, 7) factor 0.15
  - [x] interval = 10 → 两段 factor
  - [x] interval = 50 → 三段 factor
  - [x] max_ivl clamp
  - [x] 固定种子输出可复现
- [x] `scheduler_test.go` — ~25 用例：
  - [x] NewScheduler 参数越界 → error
  - [x] NewScheduler 默认配置 → 正常
  - [x] Learning 首次 Again/Hard/Good/Easy → 验证 S, D, State, Step, Due
  - [x] Learning 同日复习 → shortTerm 分支
  - [x] Learning 跨日复习 → nextStability 分支
  - [x] Learning Good 最后一步 → Review
  - [x] Learning Easy → 直接 Review
  - [x] Learning Hard step=0 len=1 → 1.5x
  - [x] Learning Hard step=0 len≥2 → 平均值
  - [x] Learning empty steps → 直接 Review
  - [x] Learning step≥len → 直接 Review
  - [x] Review 跨日 Hard/Good/Easy → nextInterval
  - [x] Review 同日 → shortTerm
  - [x] Review Again → Relearning (step=0)
  - [x] Review Again + empty relearning_steps → nextInterval
  - [x] Relearning Again/Hard/Good/Easy（对称验证）
  - [x] DisableFuzzing=false + Review 状态 → interval 不同
  - [x] DisableFuzzing=true → interval 不变
  - [x] Retrievability LastReview=nil → 0
  - [x] Retrievability 正常值

**交付标准**：ReviewCard 可跑完整复习流程，覆盖率 100%。

---

### v0.3.0 — 序列对齐 + 辅助 API

> 目标：与 py-fsrs 逐步对齐，确认算法实现正确无误。

**源码**

- [x] `scheduler.go` 追加：
  - [x] PreviewCard(card, now) → map[Rating]Card
  - [x] RescheduleCard(card, logs) → (Card, error)
  - [x] MarshalJSON() → ([]byte, error)（实现 json.Marshaler）
  - [x] UnmarshalJSON(data) → error（实现 json.Unmarshaler，重建内部预计算状态）

**测试（L7–L8）**

- [x] `scheduler_test.go` 追加（~15 用例）：
  - [x] **Scenario 1**：NewCard → Good → 3d → Good → 7d → Good
  - [x] **Scenario 2**：NewCard → Again → 同日 Good → 同日 Good
  - [x] **Scenario 3**：NewCard → Good → Good → (Review) Again → Relearning → Good → Review
  - [x] **Scenario 4**：NewCard → Easy（直接跳 Review）
  - [x] **Scenario 5**：空 learning_steps → Hard 直接进 Review
  - [x] 每个 Scenario 逐步验证 (State, Step, S, D, Due) 对齐 py-fsrs 预期值
  - [x] PreviewCard 返回 4 个 key
  - [x] RescheduleCard 正常重放
  - [x] RescheduleCard CardID 不匹配 → errors.Is(err, ErrCardIDMismatch)
  - [x] json.Marshal → json.Unmarshal 往返一致（Scheduler 配置 + 内部状态）
  - [x] json.Unmarshal 畸形数据 → error

**测试数据**

- [x] `testdata/py_fsrs_alignment.json` — py-fsrs 运行 5 个 Scenario 的预期输出
  - [x] 生成脚本：`scripts/gen_alignment_data.py`

**交付标准**：5 个序列场景全部对齐 py-fsrs（精度 1e-4），覆盖率 100%。

---

### v0.4.0 — Optimizer 基础

> 目标：dataset + loss + adam 模块可独立测试。还不能跑完整优化循环。

**源码**

- [x] `optimizer/dataset.go` — ReviewLog → 训练数据
  - [x] formatRevlogs(logs) → map[int64][]review（按 card_id 分组，组内按时间排序）
  - [x] countCrossDayReviews(data) → int
- [x] `optimizer/loss.go` — BCE + 数值梯度
  - [x] bceLoss(rPred, y) → float64
  - [x] computeBatchLoss(params, data) → float64
  - [x] numericalGradient(params, data) → [21]float64
- [x] `optimizer/adam.go` — Adam 优化器 + Cosine Annealing
  - [x] Adam 结构体：m, v [21]float64, β1, β2, ε, step
  - [x] NewAdam(lr) → *Adam
  - [x] Adam.Update(params, grads) → [21]float64
  - [x] CosineAnnealing 结构体
  - [x] NewCosineAnnealing(lrMax, tMax) → *CosineAnnealing
  - [x] CosineAnnealing.Step() → float64

**测试（L9 部分）**

- [x] `optimizer/dataset_test.go` — ~8 用例：
  - [x] 空日志 → 空 map
  - [x] 单卡多次复习 → 按时间排序
  - [x] 多卡 → 按 card_id 分组
  - [x] countCrossDayReviews 统计正确
- [x] `optimizer/loss_test.go` — ~10 用例：
  - [x] bceLoss(0.9, 1) ≈ 0.1054
  - [x] bceLoss(0.9, 0) ≈ 2.3026
  - [x] bceLoss(0.5, 1) ≈ 0.6931
  - [x] bceLoss 边界：rPred=0.001, rPred=0.999
  - [x] computeBatchLoss 对已知数据的期望值
  - [x] numericalGradient 方向正确（与解析解比较简单函数）
  - [x] numericalGradient 对称性（改变单个参数只影响该维度）
- [x] `optimizer/adam_test.go` — ~8 用例：
  - [x] 单步更新方向正确（负梯度方向）
  - [x] 多步更新后参数变化幅度合理
  - [x] bias correction 生效（前几步 m̂, v̂ 放大）
  - [x] CosineAnnealing t=0 → lr_max
  - [x] CosineAnnealing t=T_max → lr ≈ 0
  - [x] CosineAnnealing t=T_max/2 → lr ≈ lr_max/2

**交付标准**：三个 Optimizer 子模块 100% 覆盖。

---

### v0.5.0 — Optimizer 完整

> 目标：可从复习日志优化出 FSRS 参数。

**源码**

- [x] `optimizer/optimizer.go` — Optimizer 核心
  - [x] OptimizerConfig + 默认值
  - [x] NewOptimizer(cfg) → *Optimizer
  - [x] ComputeOptimalParameters(ctx, logs) → ([21]float64, error)
    - [x] 日志预处理
    - [x] num_reviews < MiniBatchSize → 返回 DefaultParameters + ErrInsufficientData
    - [x] 5 epoch 训练循环（每轮检查 ctx.Err()）
    - [x] mini-batch 数值梯度 → Adam 更新 → Clamp
    - [x] best_params 追踪
  - [x] ComputeBatchLoss(params, logs) → float64（公共封装）

**测试（L9–L10）**

- [x] `optimizer/optimizer_test.go` — ~12 用例：
  - [x] 空日志 → error
  - [x] 日志不足 MiniBatchSize → 返回 DefaultParameters
  - [x] 合成数据 2000 条（DefaultParameters 生成）→ 优化后 loss 下降
  - [x] 合成数据 → 优化后各参数偏差 < 10%
  - [x] 不同 Epochs 值 → 更多 epoch loss 更低
  - [x] 参数始终在 [LowerBounds, UpperBounds] 范围内

**交付标准**：ComputeOptimalParameters 可用，合成数据收敛测试通过。

---

### v0.6.0 — 最优保留率 + 集成测试

> 目标：Optimizer 完整功能。用真实数据验证。

**源码**

- [x] `optimizer/retention.go` — 最优保留率
  - [x] computeProbsAndCosts(logs) → map[string]float64
  - [x] simulateCost(retention, params, probsAndCosts) → float64
  - [x] ComputeOptimalRetention(ctx, params, logs) → (float64, error)
    - [x] 校验 ≥ 512 条，ReviewDuration 不为 nil
    - [x] 6 个候选 × 1000 卡 × 1 年蒙特卡洛（支持 ctx 取消）
    - [x] 返回 cost 最小的 retention

**测试（L9–L11）**

- [x] `optimizer/retention_test.go` — ~8 用例：
  - [x] 日志 < 512 → error
  - [x] ReviewDuration=nil → error
  - [x] 正常数据 → 输出 ∈ [0.70, 0.95]
  - [x] probsAndCosts 统计正确
  - [x] simulateCost 固定种子可复现
- [x] `optimizer/integration_test.go`（`//go:build integration`）— ~4 用例：
  - [x] 从 testdata/ 加载 1 个用户日志
  - [x] ComputeOptimalParameters → 各参数偏差 < 15%（vs py-fsrs）
  - [x] ComputeBatchLoss < py-fsrs loss × 1.1
  - [x] ComputeOptimalRetention 输出合理

**测试数据**

- [x] `testdata/anki_revlogs_sample.json` — anki-revlogs-10k 中 1 个用户
- [x] `scripts/gen_optimizer_baseline.py` — py-fsrs 优化同一用户，输出 baseline JSON

**交付标准**：Optimizer 全功能可用，真实数据集成测试通过。

---

### v0.7.0 — 性能 + 示例

> 目标：性能达标，示例代码可运行。

**源码**

- [x] `examples/basic/main.go` — 创建卡片 → 复习 → 查看 due
- [x] `examples/optimizer/main.go` — 参数优化 + 最优保留率
- [x] `examples/reschedule/main.go` — 用日志重放调度

**测试（L12）**

- [x] `bench_test.go` — Scheduler 性能：
  - [x] BenchmarkReviewCard — 目标 < 500ns/op
  - [x] BenchmarkGetRetrievability — 目标 < 100ns/op
  - [x] BenchmarkPreviewCard — 目标 < 2μs/op
- [x] `optimizer/bench_test.go` — Optimizer 性能：
  - [x] BenchmarkOptimize1000 — 目标 < 2s
  - [x] BenchmarkOptimize10000 — 目标 < 15s

**性能优化（如基准未达标）**

- [x] algorithm.go 热路径内联
- [x] 减少 Scheduler 中的内存分配（预分配 Card）
- [x] Optimizer 并行化（多 card 并发前向传播）

**交付标准**：全部 benchmark 达标，examples 可直接 `go run`。

---

### v0.8.0 — 开源文档

> 目标：符合 opensource.guide 的开源项目标准。

- [x] `README.md`
  - [x] 项目简介 + Badge（CI, Coverage, Go Report, License）
  - [x] 功能特性列表
  - [x] Quick Start（安装 + 5 行代码示例）
  - [x] API 速览（核心类型 + 方法签名）
  - [x] Optimizer 用法示例
  - [x] 性能数据
  - [x] 与 py-fsrs 的对齐说明
  - [x] Contributing 链接
  - [x] License
- [x] `CONTRIBUTING.md`
  - [x] 欢迎语
  - [x] 开发环境搭建（Go 1.23+, make test）
  - [x] 代码风格（gofmt, golangci-lint）
  - [x] 测试要求（100% 覆盖率，对齐测试必须通过）
  - [x] PR 流程（fork → branch → test → PR）
  - [x] Issue 规范
  - [x] Commit 规范（Conventional Commits）
- [ ] `CODE_OF_CONDUCT.md` — Contributor Covenant v2.1（deferred to v1.0.0）
- [ ] `SECURITY.md` — 安全漏洞报告流程（deferred to v1.0.0）
- [x] `CHANGELOG.md` — v0.1.0 ~ v0.8.0 全部条目

**交付标准**：README 完整可读，所有社区文档到位。

---

### v0.9.0 — CI/CD + 质量门禁

> 目标：GitHub Actions 绿色，自动化质量保证。

- [x] `.github/workflows/ci.yml`
  - [x] Go 1.23 + latest
  - [x] `go vet ./...`
  - [x] `golangci-lint run`
  - [x] `go test ./... -cover -coverprofile=coverage.out`
  - [x] 覆盖率 100% 门禁（解析 coverage.out，低于 100% fail）
  - [x] `go test -race ./...`
  - [x] 集成测试（仅 main 分支，`-tags integration`）
- [x] `.github/workflows/release.yml`
  - [x] tag push 触发
  - [x] Go module 验证
  - [x] GitHub Release 创建
- [x] `.github/ISSUE_TEMPLATE/bug_report.md`
- [x] `.github/ISSUE_TEMPLATE/feature_request.md`
- [x] `.github/PULL_REQUEST_TEMPLATE.md`
- [x] `Makefile`
  - [x] `make test` — go test ./... -cover
  - [x] `make cover` — 覆盖率报告
  - [x] `make lint` — golangci-lint run
  - [x] `make bench` — go test -bench ./...
  - [x] `make examples` — 编译全部 examples

**交付标准**：CI 全绿，PR 门禁生效。

---

### v1.0.0 — 稳定发布

> 目标：API 冻结，生产就绪，提交 awesome-fsrs。

**发布前检查清单**

- [x] 所有公共 API 稳定，无计划中的 breaking change
- [x] `go test ./...` 全部通过
- [x] `go test -cover ./...` 输出 100%
- [ ] `golangci-lint run` 零 warning（CI 中执行）
- [x] `go vet ./...` 通过
- [x] `go test -race ./...` 通过
- [x] L4 公式对齐测试全部通过（精度 1e-4）
- [x] L7 五个序列场景全部通过
- [x] L10 Optimizer 收敛测试通过
- [x] L11 集成测试通过（Go loss 0.3491 vs py-fsrs 0.3496）
- [x] BenchmarkReviewCard < 500ns/op（189ns）
- [x] BenchmarkGetRetrievability < 100ns/op（25ns）
- [x] godoc 注释覆盖所有导出符号
- [x] README / CONTRIBUTING / CHANGELOG 完整（CODE_OF_CONDUCT / SECURITY deferred）
- [x] examples/ 包含 3 个可运行示例
- [ ] GitHub Actions CI 全绿（推送后验证）
- [x] CHANGELOG.md 包含 v1.0.0 条目

**发布任务**

- [ ] `git tag v1.0.0` + push
- [ ] GitHub Release v1.0.0（含 Release Notes）
- [ ] 验证 `go get github.com/sky-flux/flux@v1.0.0` 可用
- [ ] 提交 PR 到 awesome-fsrs：`Go: Scheduler (v6) + Optimizer: flux`
- [ ] 在 FSRS 社区（GitHub Discussions / Reddit r/Anki）发布公告

---

## 项目文件清单

### 仓库根目录

| 文件 | 用途 |
|------|------|
| `LICENSE` | MIT 许可证全文 |
| `README.md` | 项目介绍、Quick Start、Badge |
| `CONTRIBUTING.md` | 贡献指南 |
| `CODE_OF_CONDUCT.md` | Contributor Covenant v2.1 |
| `CHANGELOG.md` | 语义版本变更记录 |
| `SECURITY.md` | 安全漏洞报告流程 |
| `flux.md` | 设计文档（本文件） |
| `go.mod` | Go module |
| `go.sum` | 依赖锁定 |
| `.gitignore` | Go + IDE 忽略规则 |
| `Makefile` | test / cover / lint / bench |

### GitHub 配置

| 文件 | 用途 |
|------|------|
| `.github/workflows/ci.yml` | CI：lint + test + 覆盖率门禁 |
| `.github/workflows/release.yml` | tag 触发发布 |
| `.github/ISSUE_TEMPLATE/bug_report.md` | Bug 报告模板 |
| `.github/ISSUE_TEMPLATE/feature_request.md` | 功能请求模板 |
| `.github/PULL_REQUEST_TEMPLATE.md` | PR 模板 |

### 源代码

| 文件 | 职责 |
|------|------|
| `doc.go` | 包级 godoc 注释 |
| `rating.go` | Rating 枚举 + Stringer/JSON/Text 接口 |
| `state.go` | State 枚举 + Stringer/JSON/Text 接口 |
| `card.go` | Card（含 JSON tags）、NewCard()、clone() |
| `review_log.go` | ReviewLog（含 JSON tags） |
| `errors.go` | 包级哨兵错误 |
| `parameters.go` | DefaultParameters, 边界, ValidateParameters() |
| `algorithm.go` | 纯数学函数 |
| `fuzz.go` | Fuzz 逻辑 |
| `scheduler.go` | Scheduler 核心 + json.Marshaler/Unmarshaler |

### 测试代码

| 文件 | 测试对象 |
|------|---------|
| `rating_test.go` | Rating + JSON/Text 序列化 |
| `state_test.go` | State + JSON/Text 序列化 |
| `card_test.go` | Card + JSON 往返 |
| `review_log_test.go` | ReviewLog + JSON 往返 |
| `errors_test.go` | 哨兵错误 errors.Is |
| `parameters_test.go` | 参数校验 |
| `algorithm_test.go` | 数学函数 |
| `fuzz_test.go` | Fuzz |
| `scheduler_test.go` | 状态机 + 序列对齐 + 辅助 API |
| `bench_test.go` | 性能基准 |

### Optimizer 子包

| 文件 | 职责 |
|------|------|
| `optimizer/optimizer.go` | Optimizer + ComputeOptimalParameters |
| `optimizer/adam.go` | Adam + Cosine Annealing |
| `optimizer/loss.go` | BCE + 数值梯度 |
| `optimizer/dataset.go` | ReviewLog → 训练数据 |
| `optimizer/retention.go` | ComputeOptimalRetention |
| `optimizer/optimizer_test.go` | 核心 + 收敛测试 |
| `optimizer/adam_test.go` | Adam + LR 测试 |
| `optimizer/loss_test.go` | BCE + 梯度测试 |
| `optimizer/dataset_test.go` | 数据预处理测试 |
| `optimizer/retention_test.go` | 最优保留率测试 |
| `optimizer/integration_test.go` | anki-revlogs-10k 集成测试 |
| `optimizer/bench_test.go` | Optimizer 性能基准 |

### 辅助

| 文件 | 用途 |
|------|------|
| `testdata/py_fsrs_alignment.json` | py-fsrs 序列对齐预期输出 |
| `testdata/anki_revlogs_sample.json` | anki-revlogs-10k 真实数据 |
| `scripts/gen_alignment_data.py` | 生成对齐测试数据 |
| `scripts/gen_optimizer_baseline.py` | 生成 Optimizer 基线数据 |
| `examples/basic/main.go` | 基础调度示例 |
| `examples/optimizer/main.go` | 参数优化示例 |
| `examples/reschedule/main.go` | 日志重放示例 |

---

## v5 → v6 差异清单

| 项目 | v5 | v6 |
|------|----|----|
| 参数数量 | 19 | **21** |
| DECAY | 固定 -0.5 | **-w[20]，可训练** |
| FACTOR | 固定 19/81 | **0.9^(1/DECAY) - 1** |
| 同日复习 S' | S · e^(w17·(G-3+w18)) | + **S^(-w19)** |
| 遗忘后 S' | 仅长期公式 | **min(long, short)** |
| 难度均值回归目标 | D₀(Easy) | D₀(Easy)（同 v5，v4 用 D₀(Good)） |

---

## 参考文献

1. **py-fsrs** — https://open-spaced-repetition.github.io/py-fsrs/fsrs.html
2. **go-fsrs v3** — https://pkg.go.dev/github.com/open-spaced-repetition/go-fsrs/v3
3. **riff (SiYuan)** — https://github.com/siyuan-note/riff
4. **The Algorithm** — https://github.com/open-spaced-repetition/fsrs4anki/wiki/The-Algorithm
5. **The mechanism of optimization** — https://github.com/open-spaced-repetition/fsrs4anki/wiki/The-mechanism-of-optimization
6. **anki-revlogs-10k** — https://huggingface.co/datasets/open-spaced-repetition/anki-revlogs-10k
7. **Open Source Guides** — https://opensource.guide/
8. **Personas and Pathways** — https://mozillascience.github.io/working-open-workshop/personas_pathways

---

## 依赖

| 包 | 范围 | 用途 |
|----|------|------|
| Go 标准库 | Scheduler | math, time, encoding/json, math/rand |
| Go 标准库 | Optimizer | math, math/rand, sort, context（不需要 gonum，使用纯标准库实现数值微分和 Adam 优化器） |

## 许可证

MIT
