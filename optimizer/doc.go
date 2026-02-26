// Package optimizer trains FSRS v6 parameters from historical review logs.
//
// It provides two main capabilities:
//
//   - [Optimizer.ComputeOptimalParameters] trains the 21 FSRS parameters
//     using mini-batch gradient descent with the [Adam] optimizer and
//     [CosineAnnealing] learning rate schedule. Gradients are computed via
//     numerical central differences on binary cross-entropy loss.
//
//   - [Optimizer.ComputeOptimalRetention] finds the desired retention value
//     that minimizes total review cost via Monte Carlo simulation.
//
// # Usage
//
//	opt := optimizer.NewOptimizer(optimizer.OptimizerConfig{})
//	params, err := opt.ComputeOptimalParameters(ctx, logs)
//	retention, err := opt.ComputeOptimalRetention(ctx, params, logs)
//
// # Data Requirements
//
// Parameter optimization requires enough cross-day reviews (at least
// MiniBatchSize, default 512). Optimal retention additionally requires
// ReviewDuration to be set on all review logs.
package optimizer
