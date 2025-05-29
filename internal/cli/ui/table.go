package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// TableOptions represents options for table rendering
type TableOptions struct {
	Headers           []string
	Rows              [][]string
	Footer            []string
	Alignment         []int
	AutoWrapText      bool
	AutoMergeCells    bool
	RowLine           bool
	ColumnSeparator   string
	CenterSeparator   string
	RowSeparator      string
	HeaderLine        bool
	Border            bool
	TablePadding      string
	TablePaddingLeft  int
	TablePaddingRight int
}

// DefaultTableOptions returns default table options
func DefaultTableOptions() *TableOptions {
	return &TableOptions{
		AutoWrapText:      false,
		AutoMergeCells:    false,
		RowLine:           false,
		ColumnSeparator:   "│",
		CenterSeparator:   "┼",
		RowSeparator:      "─",
		HeaderLine:        true,
		Border:            true,
		TablePadding:      " ",
		TablePaddingLeft:  1,
		TablePaddingRight: 1,
	}
}

// RenderTable renders a table with the given options
func RenderTable(opts *TableOptions) {
	table := tablewriter.NewWriter(os.Stdout)

	// Set headers
	if len(opts.Headers) > 0 {
		table.Header(interfaceSlice(opts.Headers)...)
	}

	// Set footer
	if len(opts.Footer) > 0 {
		table.Footer(interfaceSlice(opts.Footer)...)
	}

	// Add rows
	for _, row := range opts.Rows {
		if err := table.Append(interfaceSlice(row)...); err != nil {
			fmt.Printf("Error appending row: %v\n", err)
		}
	}

	// Render the table
	if err := table.Render(); err != nil {
		fmt.Printf("Error rendering table: %v\n", err)
	}
}

// Helper function to convert []string to []interface{}
func interfaceSlice(slice []string) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}

// RenderSQLTable renders a table formatted for SQL results
func RenderSQLTable(headers []string, rows [][]string) {
	opts := DefaultTableOptions()
	opts.Headers = headers
	opts.Rows = rows
	opts.RowLine = true

	fmt.Println()
	SQLKeyword.Println("Query Results:")
	RenderTable(opts)

	if len(rows) > 0 {
		Muted.Printf("\n%d row(s) returned\n", len(rows))
	} else {
		Warning.Println("\nNo rows returned")
	}
}

// RenderK8sTable renders a table formatted for Kubernetes resources
func RenderK8sTable(resourceType string, headers []string, rows [][]string) {
	opts := DefaultTableOptions()
	opts.Headers = headers
	opts.Rows = rows

	fmt.Println()
	K8sResource.Printf("%s:\n", strings.ToUpper(resourceType))
	RenderTable(opts)
}

// RenderDockerTable renders a table formatted for Docker resources
func RenderDockerTable(resourceType string, headers []string, rows [][]string) {
	opts := DefaultTableOptions()
	opts.Headers = headers
	opts.Rows = rows

	fmt.Println()
	DockerImage.Printf("DOCKER %s:\n", strings.ToUpper(resourceType))
	RenderTable(opts)
}

// RenderKeyValueTable renders a simple key-value table
func RenderKeyValueTable(title string, data map[string]string) {
	if title != "" {
		Header.Println(title)
		fmt.Println()
	}

	maxKeyLen := 0
	for key := range data {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}

	for key, value := range data {
		Label.Printf("%-*s : ", maxKeyLen, key)
		Value.Println(value)
	}
}
