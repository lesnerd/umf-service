package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type DataGenerator struct {
	switchCount    int
	dataCache      []byte
	cacheTime      time.Time
	mutex          sync.RWMutex
	generating     bool
	stopChan       chan struct{}
	generationID   int64
	lastGeneration time.Time
	ready          bool
}

var generator *DataGenerator

func main() {
	port := getEnv("GENERATOR_PORT", "9001")
	switchCount := parseInt(getEnv("GENERATOR_SWITCH_COUNT", "1000"))

	fmt.Printf("Starting eagerptive CSV generator on port %s with %d switches\n", port, switchCount)

	// Init eager data generator
	generator = &DataGenerator{
		switchCount: switchCount,
		stopChan:    make(chan struct{}),
		ready:       false,
	}

	// Generate initial data immediately and wait for it (blocking)
	fmt.Println("Generating initial data...")
	generator.generateData()
	generator.ready = true
	fmt.Printf("Initial data ready! Generated %d bytes\n", len(generator.dataCache))

	// Start eager data generation in background
	go generator.startDataGeneration()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Serve pre-generated CSV data
	router.GET("/counters", func(c *gin.Context) {
		handleCounters(c)
	})

	// Just to make sure the service is up
	router.GET("/health", func(c *gin.Context) {
		generator.mutex.RLock()
		status := "ready"
		if !generator.ready {
			status = "initializing"
		}
		generator.mutex.RUnlock()

		c.JSON(http.StatusOK, gin.H{
			"status":           status,
			"switches":         switchCount,
			"timestamp":        time.Now().Format(time.RFC3339),
			"last_generation":  generator.lastGeneration.Format(time.RFC3339),
			"data_ready":       generator.ready,
			"cache_size_bytes": len(generator.dataCache),
		})
	})

	fmt.Printf("Generator service ready on port %s\n", port)
	router.Run(":" + port)
}

// Generate continuously data in the background
func (dg *DataGenerator) startDataGeneration() {
	// Generate new data every 10 seconds (matching the UFM poll interval)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	fmt.Println("Background data generation started (every 10 seconds)")

	for {
		select {
		case <-dg.stopChan:
			fmt.Println("Background data generation stopped")
			return
		case <-ticker.C:
			dg.generateData()
		}
	}
}

// generateData creates new CSV data and caches it
func (dg *DataGenerator) generateData() {
	dg.mutex.Lock()
	dg.generating = true
	dg.mutex.Unlock()

	now := time.Now()
	dg.generationID = now.UnixNano()

	estimatedSize := 100 + dg.switchCount*50*150 // ~150 chars per record
	buf := make([]byte, 0, estimatedSize)

	// Write CSV header
	buf = append(buf, "switch_id,timestamp,bandwidth_mbps,latency_ms,packet_errors,utilization_pct,temperature_c\n"...)

	// Random values to avoid repeated rand calls
	randomValues := make([]float64, dg.switchCount*100*6) // 6 values per switch * up to 100 records
	for i := range randomValues {
		randomValues[i] = rand.Float64()
	}
	randomIndex := 0

	// Generate data for each switch with staggered timestamps
	for i := 1; i <= dg.switchCount; i++ {
		switchID := fmt.Sprintf("switch-%03d", i)
		messagesPerSwitch := rand.Intn(100) + 1 // 1 to 100 messages per switch

		for j := 0; j < messagesPerSwitch; j++ {
			// Use pre-generated random values
			if randomIndex >= len(randomValues) {
				// Regenerate if we run out of values
				for k := range randomValues {
					randomValues[k] = rand.Float64()
				}
				randomIndex = 0
			}

			// Stagger timestamps by 10-50ms for each message
			staggerMs := 10 + int(randomValues[randomIndex]*40)
			timestamp := now.Add(time.Duration((i-1)*100+j*staggerMs) * time.Millisecond)
			randomIndex++

			// Generate realistic fake metrics using pre-generated values
			bandwidth := 100 + randomValues[randomIndex]*800 // 100-900 Mbps
			randomIndex++
			latency := 0.5 + randomValues[randomIndex]*5.0 // 0.5-5.5 ms
			randomIndex++
			packetErrors := int(randomValues[randomIndex] * 10) // 0-9 errors
			randomIndex++
			utilization := 10 + randomValues[randomIndex]*80 // 10-90%
			randomIndex++
			temperature := 30 + randomValues[randomIndex]*30 // 30-60°C
			randomIndex++

			// Use optimized string building for maximum performance
			record := fmt.Sprintf("%s,%s,%.2f,%.3f,%d,%.2f,%.2f\n",
				switchID,
				timestamp.Format(time.RFC3339Nano),
				bandwidth,
				latency,
				packetErrors,
				utilization,
				temperature,
			)
			buf = append(buf, record...)
		}
	}

	// Update cache atomically
	dg.mutex.Lock()
	dg.dataCache = make([]byte, len(buf))
	copy(dg.dataCache, buf)
	dg.cacheTime = now
	dg.lastGeneration = now
	dg.generating = false
	dg.ready = true
	dg.mutex.Unlock()

	fmt.Printf("Generated new batch: %d bytes, %d switches, generation_id=gen_%d at %s\n",
		len(buf), dg.switchCount, dg.generationID, now.Format(time.RFC3339))
}

// handleCounters serves pre-generated CSV data immediately
func handleCounters(c *gin.Context) {
	generator.mutex.RLock()
	defer generator.mutex.RUnlock()

	// Check if cached data is ready
	if !generator.ready || len(generator.dataCache) == 0 {
		// No cached data available, return service unavailable
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "No pre-generated data available",
			"ready": generator.ready,
		})
		return
	}

	// Set CSV headers
	c.Header("Content-Type", "text/csv")
	c.Header("X-Generation-ID", fmt.Sprintf("gen_%d", generator.generationID))
	c.Header("X-Data-Timestamp", generator.cacheTime.Format(time.RFC3339Nano))
	c.Header("X-Switch-Count", strconv.Itoa(generator.switchCount))
	c.Header("X-Pre-Generated", "true")
	c.Header("X-Data-Size", strconv.Itoa(len(generator.dataCache)))
	c.Header("X-Last-Generation", generator.lastGeneration.Format(time.RFC3339))

	// Serve the pre-generated data immediately mocking the UFM API
	c.Data(http.StatusOK, "text/csv", generator.dataCache)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string) int {
	if s == "" {
		return 10
	}

	if result, err := strconv.Atoi(s); err == nil {
		return result
	}
	return 10
}
