-- Drop unused tick dataset tables (dead code: tick_dataset_repository.go removed, no callers).
DROP TABLE IF EXISTS tick_dataset_ticks;
DROP TABLE IF EXISTS tick_datasets;
