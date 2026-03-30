from app.inference.aggregator import SessionAggregator


def test_summary_ready_rule() -> None:
    agg = SessionAggregator()
    agg.update_session(['red'] * 7 + ['green'] * 2 + ['half'], list(range(10)))
    summary = agg.build_summary()
    assert summary.total_detected == 10
    assert summary.harvest_suggestion == 'ready'


def test_summary_partially_ready_rule() -> None:
    agg = SessionAggregator()
    agg.update_session(['half'] * 5 + ['green'] * 5, list(range(10)))
    summary = agg.build_summary()
    assert summary.harvest_suggestion == 'partially_ready'


def test_deduplicate_by_track_id() -> None:
    agg = SessionAggregator()
    agg.update_session(['half'], [1])
    agg.update_session(['half'], [1])
    summary = agg.build_summary()
    assert summary.total_detected == 1
