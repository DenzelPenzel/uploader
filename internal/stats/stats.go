package stats

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type Statistic struct {
	mutex           sync.RWMutex
	shutdown        chan struct{}
	Hostname        string
	StartTime       time.Time
	ProcessID       int
	ResponseCounts  map[string]int
	TotalRespCounts map[string]int
	TotalRespTime   time.Duration
	TotalRespSize   int64
	MetricCounts    map[string]int
	MetricTimers    map[string]time.Duration
}

type MetricLabel struct {
	Name  string
	Value string
}

func NewStatistic() *Statistic {
	hostname, _ := os.Hostname()

	statistic := &Statistic{
		shutdown:        make(chan struct{}, 1),
		StartTime:       time.Now(),
		ProcessID:       os.Getpid(),
		ResponseCounts:  make(map[string]int),
		TotalRespCounts: make(map[string]int),
		Hostname:        hostname,
	}

	go statistic.resetResponseCountsPeriodically()

	return statistic
}

func (stat *Statistic) Close() {
	close(stat.shutdown)
}

func (stat *Statistic) resetResponseCountsPeriodically() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stat.shutdown:
			return
		case <-ticker.C:
			stat.resetResponseCounts()
		}
	}
}

func (stat *Statistic) resetResponseCounts() {
	stat.mutex.Lock()
	defer stat.mutex.Unlock()
	stat.ResponseCounts = make(map[string]int)
}

func (stat *Statistic) WrapHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime, recorder := stat.StartRecording(w)

		handler.ServeHTTP(recorder, r)

		stat.EndRecording(startTime, recorder)
	})
}

func (stat *Statistic) StartRecording(writer http.ResponseWriter) (time.Time, ResponseWriter) {
	return time.Now(), NewResponseRecorder(writer, http.StatusOK)
}

func (stat *Statistic) EndRecording(startTime time.Time, recorder ResponseWriter) {
	duration := time.Since(startTime)

	stat.mutex.Lock()
	defer stat.mutex.Unlock()

	if status := recorder.Status(); status != 0 {
		statusCode := fmt.Sprintf("%d", status)
		stat.ResponseCounts[statusCode]++
		stat.TotalRespCounts[statusCode]++
		stat.TotalRespTime += duration
		stat.TotalRespSize += int64(recorder.Size())
	}
}

func (stat *Statistic) RecordMetric(metric string, startTime time.Time, labels []MetricLabel) {
	labels = append(labels, MetricLabel{"host", stat.Hostname})
	duration := time.Since(startTime)

	stat.mutex.Lock()
	defer stat.mutex.Unlock()

	stat.MetricCounts[metric]++
	stat.MetricTimers[metric] += duration
}

type StatisticData struct {
	ProcessID              int                `json:"pid"`
	Hostname               string             `json:"hostname"`
	UpTime                 string             `json:"uptime"`
	UpTimeSec              float64            `json:"uptime_sec"`
	Time                   string             `json:"time"`
	TimeUnix               int64              `json:"unixtime"`
	StatusCodeCount        map[string]int     `json:"status_code_count"`
	TotalStatusCodeCount   map[string]int     `json:"total_status_code_count"`
	ResponseCount          int                `json:"count"`
	TotalResponseCount     int                `json:"total_count"`
	TotalResponseTime      string             `json:"total_response_time"`
	TotalResponseTimeSec   float64            `json:"total_response_time_sec"`
	TotalResponseSize      int64              `json:"total_response_size"`
	AverageResponseSize    int64              `json:"average_response_size"`
	AverageResponseTime    string             `json:"average_response_time"`
	AverageResponseTimeSec float64            `json:"average_response_time_sec"`
	TotalMetricCounts      map[string]int     `json:"total_metrics_counts"`
	AverageMetricTimes     map[string]float64 `json:"average_metrics_timers"`
}

func (stat *Statistic) GatherData() *StatisticData {
	stat.mutex.RLock()
	defer stat.mutex.RUnlock()

	responseCounts := make(map[string]int, len(stat.ResponseCounts))
	totalResponseCounts := make(map[string]int, len(stat.TotalRespCounts))
	totalMetricCounts := make(map[string]int, len(stat.MetricCounts))
	metricTimes := make(map[string]float64, len(stat.MetricCounts))

	currentTime := time.Now()
	uptime := currentTime.Sub(stat.StartTime)

	responseCount := copyCounts(stat.ResponseCounts, responseCounts)
	totalCount := copyCounts(stat.TotalRespCounts, totalResponseCounts)

	avgResponseTime, avgResponseSize := stat.calculateAverages(totalCount)

	for metric, count := range stat.MetricCounts {
		totalMetricTime := stat.MetricTimers[metric]
		metricTimes[metric] = (totalMetricTime / time.Duration(count)).Seconds()
		totalMetricCounts[metric] = count
	}

	return &StatisticData{
		ProcessID:              stat.ProcessID,
		UpTime:                 uptime.String(),
		UpTimeSec:              uptime.Seconds(),
		Time:                   currentTime.String(),
		TimeUnix:               currentTime.Unix(),
		StatusCodeCount:        responseCounts,
		TotalStatusCodeCount:   totalResponseCounts,
		ResponseCount:          responseCount,
		TotalResponseCount:     totalCount,
		TotalResponseTime:      stat.TotalRespTime.String(),
		TotalResponseSize:      stat.TotalRespSize,
		TotalResponseTimeSec:   stat.TotalRespTime.Seconds(),
		TotalMetricCounts:      totalMetricCounts,
		AverageResponseSize:    avgResponseSize,
		AverageResponseTime:    avgResponseTime.String(),
		AverageResponseTimeSec: avgResponseTime.Seconds(),
		AverageMetricTimes:     metricTimes,
	}
}

func (stat *Statistic) calculateAverages(count int) (time.Duration, int64) {
	if count == 0 {
		return 0, 0
	}
	return stat.TotalRespTime / time.Duration(count), stat.TotalRespSize / int64(count)
}

func copyCounts(src, dest map[string]int) (sum int) {
	// note: for the map no need to use pointers bcs map are ref type
	for key, count := range src {
		dest[key] = count
		sum += count
	}
	return sum
}
