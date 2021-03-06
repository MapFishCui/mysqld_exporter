// Scrape `performance_schema.file_summary_by_instance`.

package collector

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"

	"flag"
	"path"
	"path/filepath"
	"strings"
)

const perfFileInstancesQuery = `
	SELECT
	    FILE_NAME,
	    COUNT_READ, COUNT_WRITE,
	    SUM_NUMBER_OF_BYTES_READ, SUM_NUMBER_OF_BYTES_WRITE
	  FROM performance_schema.file_summary_by_instance
	     where FILE_NAME REGEXP ?
	`

// Metric descriptors.
var (
	performanceSchemaFileInstancesFilter = flag.String(
		"collect.perf_schema.file_instances.filter", ".*",
		"RegEx file_name filter for performance_schema.file_summary_by_instance",
	)

	performanceSchemaFileInstancesRemovePrefix = flag.Bool(
		"collect.perf_schema.file_instances.remove_prefix", true,
		"Remove path prefix in performance_schema.file_summary_by_instance",
	)

	performanceSchemaFileInstancesBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, performanceSchema, "file_instances_bytes"),
		"The number of bytes processed by file read/write operations.",
		[]string{"db_name", "file_name", "mode"}, nil,
	)
	performanceSchemaFileInstancesCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, performanceSchema, "file_instances_total"),
		"The total number of file read/write operations.",
		[]string{"db_name", "file_name", "mode"}, nil,
	)
)

// ScrapePerfFileEvents collects from `performance_schema.file_summary_by_event_name`.
func ScrapePerfFileInstances(db *sql.DB, ch chan <- prometheus.Metric) error {
	// Timers here are returned in picoseconds.
	perfSchemaFileInstancesRows, err := db.Query(perfFileInstancesQuery, *performanceSchemaFileInstancesFilter)
	if err != nil {
		return err
	}
	defer perfSchemaFileInstancesRows.Close()

	var (
		fileName, dbName string
		countRead, countWrite uint64
		sumBytesRead, sumBytesWritten uint64
	)

	for perfSchemaFileInstancesRows.Next() {
		if err := perfSchemaFileInstancesRows.Scan(
			&fileName,
			&countRead, &countWrite,
			&sumBytesRead, &sumBytesWritten,
		); err != nil {
			return err
		}

		if *performanceSchemaFileInstancesRemovePrefix {
			path_tree := strings.Split(path.Dir(filepath.ToSlash(fileName)), "/")
			fileName = path.Base(filepath.ToSlash(fileName))
			dbName = path_tree[len(path_tree) - 1]
		}
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaFileInstancesCountDesc, prometheus.CounterValue, float64(countRead),
			dbName, fileName, "read",
		)
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaFileInstancesCountDesc, prometheus.CounterValue, float64(countWrite),
			dbName, fileName, "write",
		)
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaFileInstancesBytesDesc, prometheus.CounterValue, float64(sumBytesRead),
			dbName, fileName, "read",
		)
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaFileInstancesBytesDesc, prometheus.CounterValue, float64(sumBytesWritten),
			dbName, fileName, "write",
		)

	}
	return nil
}
