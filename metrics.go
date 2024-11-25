package queue

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	QueueMetricsSubsystem = "queue_observer"
)

type QueueMetrics struct {
	ObserverSubscribed     *prometheus.GaugeVec
	ObserverReadCount      *prometheus.CounterVec
	ObserverWriteCount     *prometheus.CounterVec
	ObserverWriteFullCount *prometheus.CounterVec
}

func NewQueueMetrics() *QueueMetrics {
	// Initialize Prometheus metrics
	var ObserverSubscribed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: QueueMetricsSubsystem,
			Name:      "subscribers_gauge",
			Help:      "Number of event observer subscribers",
		},
		[]string{"queue"},
	)

	var ObserverReadCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: QueueMetricsSubsystem,
			Name:      "event_read_count",
			Help:      "Number of events received by observer",
		},
		[]string{"queue"},
	)

	var ObserverWriteCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: QueueMetricsSubsystem,
			Name:      "event_write_count",
			Help:      "Number of events sent by observer",
		},
		[]string{"queue"},
	)

	var ObserverWriteFullCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: QueueMetricsSubsystem,
			Name:      "event_write_skip_count",
			Help:      "Number of events skipped by observer cause chan is full",
		},
		[]string{"queue"},
	)

	prometheus.MustRegister(ObserverSubscribed)
	prometheus.MustRegister(ObserverReadCount)
	prometheus.MustRegister(ObserverWriteCount)
	prometheus.MustRegister(ObserverWriteFullCount)

	return &QueueMetrics{
		ObserverSubscribed:     ObserverSubscribed,
		ObserverReadCount:      ObserverReadCount,
		ObserverWriteCount:     ObserverWriteCount,
		ObserverWriteFullCount: ObserverWriteFullCount,
	}
}
