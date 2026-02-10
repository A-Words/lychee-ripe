# Data Specification

## Label classes
- `0`: green
- `1`: half
- `2`: red
- `3`: young

## Annotation rules
- Every visible lychee should have one bounding box.
- Each bounding box must include exactly one ripeness label.
- Occluded lychees over 60% should be ignored.
- Blurry objects smaller than 12x12 pixels should be ignored.

## Split strategy
- Train/Val/Test = 70/15/15
- Keep scene-level split to avoid leakage from adjacent frames.
- Freeze a `golden_test_set` that is never used in training.

## Quality gates
- 5% random double-annotation sample for consistency check.
- Generate class distribution report each data refresh.
- Maintain bad-case list for hard examples (low light, blur, cluster occlusion).
