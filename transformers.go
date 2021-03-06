package main

import (
	"errors"
	"fmt"
	"strings"
)

// Transformer functions represent a transformation from (one or many) Row
// pointers to (one or many) Row pointers. A Transformer should close its output
// channel when finished with it.
type Transformer func(input <-chan *Row, output chan<- *Row)

// Filter is a function that takes a Row pointer and indicates whether it is to
// be accepted or excluded
type Filter func(*Row) bool

// RowSkipper returns a transformation that skips *n* rows
func RowSkipper(n int) Transformer {
	return func(input <-chan *Row, output chan<- *Row) {
		count := 0
		for row := range input {
			if count >= n {
				output <- row
			}
			count++
		}
		close(output)
	}
}

// RowLimiter returns a transformation that stops after *n* rows
func RowLimiter(n int) Transformer {
	return func(input <-chan *Row, output chan<- *Row) {
		count := 0
		for row := range input {
			output <- row
			count++
			if count == n {
				break
			}
		}
		close(output)
	}
}

func contains(set []int, item int) bool {
	for _, i := range set {
		if i == item {
			return true
		}
	}
	return false
}

func argin(args []string, m string) (int, error) {
	for i, a := range args {
		if a == m {
			return i, nil
		}
	}
	return -1, errors.New("not found")
}

// ColumnIntSelector creates a Transformer that retains a subset of columns based on column indices
func ColumnIntSelector(indices []int) Transformer {
	return func(input <-chan *Row, output chan<- *Row) {
		var colnames, values []string
		var newRow *Row
		first := true
		for row := range input {
			if first {
				first = !first
				for _, idx := range indices {
					if idx > len(row.ColumnNames) {
						panic(fmt.Sprintf("index %d exceeds row length (%d)", idx, len(row.ColumnNames)))
					}
					colnames = append(colnames, row.ColumnNames[idx])
				}
			}
			values = make([]string, len(indices))
			for i, idx := range indices {
				values[i] = row.Values[idx]
			}
			newRow = &Row{colnames, values}
			output <- newRow
		}
		close(output)
	}
}

// ColumnStringSelector creates a Transformer that retains a subset of columns
func ColumnStringSelector(columns []string) Transformer {
	return func(input <-chan *Row, output chan<- *Row) {
		indices := make([]int, len(columns))
		row := <-input
		if row == nil {
			// There are no rows to process, so shutter
			close(output)
			return
		}

		// Determine the column indices
		for i, col := range columns {
			idx, err := argin(row.ColumnNames, col)
			if err != nil {
				panic(err)
			}
			indices[i] = idx
		}

		newInput := make(chan *Row)
		go ColumnIntSelector(indices)(newInput, output)
		newInput <- row
		for row := range input {
			newInput <- row
		}
		close(newInput)
	}
}

func predicateAsFunc(predicate string) (Filter, error) {
	parts := strings.SplitN(predicate, "=", 2)
	if len(parts) != 2 {
		return nil, errors.New("predicate parsing error")
	}
	return func(row *Row) bool {
		colIdx, err := argin(row.ColumnNames, strings.TrimSpace(parts[0]))
		if err != nil {
			panic(err)
		}
		return row.Values[colIdx] == strings.TrimSpace(parts[1])
	}, nil
}

// Predicator converts a string predicate into a function that applies it
func Predicator(predicate string) Transformer {
	filter, err := predicateAsFunc(predicate)
	if err != nil {
		// predicate unparseable
		panic(err)
	}
	return func(input <-chan *Row, output chan<- *Row) {
		for row := range input {
			if filter(row) {
				output <- row
			}
		}
		close(output)
	}
}

// IdentityTransformer passes input through unmodified
func IdentityTransformer(input <-chan *Row, output chan<- *Row) {
	for row := range input {
		output <- row
	}
	close(output)
}
