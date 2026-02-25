#!/usr/bin/env python3
"""Generate alignment test data from py-fsrs for the flux Go library.

Usage:
    pip install fsrs
    python scripts/gen_alignment_data.py > testdata/py_fsrs_alignment.json

Produces JSON with 5 scenarios, each a sequence of review steps.
Each step records the card state after the review.
"""

import json
import sys
from datetime import datetime, timedelta, timezone
from fsrs import Scheduler, Card, Rating

T0 = datetime(2025, 6, 15, 10, 0, 0, tzinfo=timezone.utc)

# Default FSRS v6 scheduler (default parameters, retention=0.9, no fuzz).
f = Scheduler(enable_fuzzing=False)


def card_to_dict(card):
    """Extract the fields we care about from a Card."""
    return {
        "state": card.state.name if hasattr(card.state, "name") else str(card.state),
        "step": card.step,
        "stability": round(float(card.stability), 6) if card.stability is not None else None,
        "difficulty": round(float(card.difficulty), 6) if card.difficulty is not None else None,
        "due": card.due.isoformat() if card.due else None,
    }


def run_scenario(name, steps):
    """Run a scenario: list of (rating, review_time) tuples."""
    card = Card()
    results = []
    for rating, review_time in steps:
        card, log = f.review_card(card, rating, review_time)
        results.append({
            "rating": rating.name,
            "review_time": review_time.isoformat(),
            "card": card_to_dict(card),
        })
    return {"name": name, "steps": results}


scenarios = []

# Scenario 1: NewCard -> Good -> 10m -> Good -> 3d -> Good
scenarios.append(run_scenario("good_good_good", [
    (Rating.Good, T0),
    (Rating.Good, T0 + timedelta(minutes=10)),
    (Rating.Good, T0 + timedelta(days=3, minutes=10)),
]))

# Scenario 2: NewCard -> Again -> same-day Good -> same-day Good
scenarios.append(run_scenario("again_good_good_sameday", [
    (Rating.Again, T0),
    (Rating.Good, T0 + timedelta(minutes=5)),
    (Rating.Good, T0 + timedelta(minutes=15)),
]))

# Scenario 3: NewCard -> Good -> Good -> (Review) Again -> Relearning -> Good -> Review
scenarios.append(run_scenario("good_good_again_relearning_good", [
    (Rating.Good, T0),
    (Rating.Good, T0 + timedelta(minutes=10)),
    (Rating.Again, T0 + timedelta(days=5, minutes=10)),
    (Rating.Good, T0 + timedelta(days=5, minutes=20)),
]))

# Scenario 4: NewCard -> Easy (skip directly to Review)
scenarios.append(run_scenario("easy_direct_review", [
    (Rating.Easy, T0),
]))

# Scenario 5: NewCard -> Hard (stays Learning with step interpolation)
scenarios.append(run_scenario("hard_from_new", [
    (Rating.Hard, T0),
]))

output = {
    "generator": "py-fsrs",
    "parameters": "default",
    "desired_retention": 0.9,
    "enable_fuzz": False,
    "scenarios": scenarios,
}

json.dump(output, sys.stdout, indent=2, default=str)
print()
