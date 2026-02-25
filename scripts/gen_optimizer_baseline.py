#!/usr/bin/env python3
"""
Generate synthetic review logs using py-fsrs Scheduler with known parameters,
then optimize with py-fsrs to produce a baseline for Go integration tests.

This approach avoids needing the gated anki-revlogs-10k dataset while still
validating that the Go optimizer converges to the same parameters as py-fsrs.

Outputs:
  testdata/anki_revlogs_sample.json       - synthetic review logs
  testdata/py_fsrs_optimizer_baseline.json - optimized params, loss, retention
"""

import json
import os
import random
from datetime import datetime, timedelta, timezone

from fsrs import Card, ReviewLog, Rating, Scheduler, Optimizer

# Use perturbed parameters so the optimizer has something to converge toward.
# Start from defaults but perturb a few key parameters.
TRUE_PARAMS = [
    0.25, 1.5, 2.5, 9.0,        # w[0..3]  slightly different initial stability
    6.4133, 0.8334, 3.0194, 0.001,  # w[4..7]  same
    1.8722, 0.1666, 0.796, 1.4835,  # w[8..11] same
    0.0614, 0.2629, 1.6483, 0.6014, # w[12..15] same
    1.8729, 0.5425, 0.0912, 0.0658, # w[16..19] same
    0.1542,                          # w[20] same
]

NUM_CARDS = 500
REVIEWS_PER_CARD = 10
SEED = 42


def generate_review_logs():
    """Generate review logs by simulating cards with TRUE_PARAMS."""
    rng = random.Random(SEED)
    scheduler = Scheduler(parameters=TRUE_PARAMS, enable_fuzzing=False)

    base_time = datetime(2024, 1, 1, 10, 0, 0, tzinfo=timezone.utc)
    logs = []

    for i in range(NUM_CARDS):
        card_id = i + 1
        card = Card(card_id=card_id, due=base_time)
        now = base_time

        for j in range(REVIEWS_PER_CARD):
            # Determine rating based on retrievability
            r = scheduler.get_card_retrievability(card=card, current_datetime=now)
            r_val = float(r) if not isinstance(r, (int, float)) else r
            recalled = rng.random() < r_val

            if not recalled:
                rating = Rating.Again
            else:
                p = rng.random()
                if p < 0.05:
                    rating = Rating.Hard
                elif p < 0.85:
                    rating = Rating.Good
                else:
                    rating = Rating.Easy

            # Random review duration (realistic: 2-15 seconds in ms)
            duration_ms = rng.randint(2000, 15000)

            logs.append(ReviewLog(
                card_id=card_id,
                rating=rating,
                review_datetime=now,
                review_duration=timedelta(milliseconds=duration_ms),
            ))

            card, _ = scheduler.review_card(
                card=card, rating=rating, review_datetime=now
            )
            now = card.due

    return logs


def main():
    print("Generating synthetic review logs...")
    review_logs = generate_review_logs()
    print(f"Generated {len(review_logs)} review logs for {NUM_CARDS} cards")

    # Convert to JSON format
    rating_to_int = {Rating.Again: 1, Rating.Hard: 2, Rating.Good: 3, Rating.Easy: 4}
    review_logs_json = []
    for log in review_logs:
        entry = {
            "card_id": log.card_id,
            "rating": rating_to_int[log.rating],
            "review_datetime": log.review_datetime.isoformat(),
        }
        if log.review_duration is not None:
            entry["review_duration_ms"] = int(log.review_duration.total_seconds() * 1000)
        review_logs_json.append(entry)

    # Save sample data
    out_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "testdata")
    os.makedirs(out_dir, exist_ok=True)

    sample_path = os.path.join(out_dir, "anki_revlogs_sample.json")
    with open(sample_path, "w") as f:
        json.dump(review_logs_json, f)
    print(f"Saved {sample_path}")

    # Run py-fsrs optimizer
    print("Running py-fsrs optimizer...")
    opt = Optimizer(review_logs)
    optimized_params = opt.compute_optimal_parameters(verbose=True)
    print(f"Optimized params: {[round(p, 4) for p in optimized_params]}")
    print(f"True params:      {TRUE_PARAMS}")

    # Compare
    for i, (opt_p, true_p) in enumerate(zip(optimized_params, TRUE_PARAMS)):
        if true_p != 0:
            pct = abs(opt_p - true_p) / abs(true_p) * 100
        else:
            pct = abs(opt_p) * 100
        marker = " ***" if pct > 15 else ""
        print(f"  w[{i:2d}]: opt={opt_p:8.4f} true={true_p:8.4f} diff={pct:5.1f}%{marker}")

    batch_loss = opt._compute_batch_loss(parameters=optimized_params)
    print(f"Batch loss (optimized): {batch_loss:.6f}")

    # Also compute loss with default params for comparison
    from fsrs.scheduler import DEFAULT_PARAMETERS
    default_loss = opt._compute_batch_loss(parameters=list(DEFAULT_PARAMETERS))
    print(f"Batch loss (default):   {default_loss:.6f}")

    print("Computing optimal retention...")
    try:
        optimal_retention = opt.compute_optimal_retention(parameters=optimized_params)
        print(f"Optimal retention: {optimal_retention}")
    except Exception as e:
        print(f"Optimal retention failed: {e}")
        optimal_retention = None

    # Save baseline
    baseline = {
        "true_parameters": TRUE_PARAMS,
        "optimized_parameters": optimized_params,
        "batch_loss": batch_loss,
        "default_loss": default_loss,
        "optimal_retention": optimal_retention,
    }
    baseline_path = os.path.join(out_dir, "py_fsrs_optimizer_baseline.json")
    with open(baseline_path, "w") as f:
        json.dump(baseline, f, indent=2)
    print(f"Saved {baseline_path}")
    print("Done!")


if __name__ == "__main__":
    main()
