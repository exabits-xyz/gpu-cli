package types

// MetricPoint is a single data point for a scalar metric (CPU, memory).
type MetricPoint struct {
	Time  int64   `json:"time"`  // Unix timestamp
	Value float64 `json:"value"` // percentage or absolute value
}

// MetricSeries is a time series for a scalar metric (CPU, memory).
type MetricSeries struct {
	Unit string        `json:"unit"`
	Data []MetricPoint `json:"data"`
}

// DiskPoint is a single data point for disk I/O.
type DiskPoint struct {
	Time  int64   `json:"time"`
	Read  float64 `json:"read"`
	Write float64 `json:"write"`
}

// DiskSeries is a time series for disk I/O metrics.
type DiskSeries struct {
	Unit string      `json:"unit"`
	Data []DiskPoint `json:"data"`
}

// NetworkPoint is a single data point for network I/O.
type NetworkPoint struct {
	Time int64   `json:"time"`
	In   float64 `json:"in"`
	Out  float64 `json:"out"`
}

// NetworkSeries is a time series for network I/O metrics.
type NetworkSeries struct {
	Unit string         `json:"unit"`
	Data []NetworkPoint `json:"data"`
}

// VMMetrics is the data object returned by GET /virtual-machines/{id}/metrics.
type VMMetrics struct {
	CPU     MetricSeries  `json:"cpu"`
	Memory  MetricSeries  `json:"memory"`
	Disk    DiskSeries    `json:"disk"`
	Network NetworkSeries `json:"network"`
}
