package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"runtime"
	"time"
)

func (h handlers) sysStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		memStats := new(runtime.MemStats)
		runtime.ReadMemStats(memStats)
		numGoroutines := runtime.NumGoroutine()
		c.JSON(http.StatusOK, gin.H{
			"time":            now.UnixNano(),
			"go_version":      runtime.Version(),
			"go_os":           runtime.GOOS,
			"go_arch":         runtime.GOARCH,
			"cpu_num":         runtime.NumCPU(),
			"goroutine_num":   runtime.NumGoroutine(),
			"go_max_procs":    runtime.GOMAXPROCS(0),
			"c_go_call_num":   runtime.NumCgoCall(),
			"mem_alloc":       memStats.Alloc,
			"mem_total_alloc": memStats.TotalAlloc,
			"mem_sys":         memStats.Sys,
			"goroutines":      numGoroutines,
		})
	}
}
