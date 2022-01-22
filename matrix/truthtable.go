package matrix

import (
	"fmt"
	"strings"
)

// TruthTable takes in n items and maintains an n x n table of booleans for each ordered pair
type TruthTable struct {
	Froms      []string
	Tos        []string
	toSet      map[string]bool
	Values     map[string]map[string]bool
	Bandwidths map[string]map[string]*ProbeJobBandwidthResults
}

// NewTruthTableFromItems creates a new truth table with items
func NewTruthTableFromItems(items []string, defaultValue *bool) *TruthTable {
	return NewTruthTable(items, items, defaultValue)
}

// NewTruthTable creates a new truth table with froms and tos
func NewTruthTable(froms, tos []string, defaultValue *bool) *TruthTable {
	values := map[string]map[string]bool{}
	bandwidths := map[string]map[string]*ProbeJobBandwidthResults{}
	for _, from := range froms {
		values[from] = map[string]bool{}
		bandwidths[from] = map[string]*ProbeJobBandwidthResults{}
		for _, to := range tos {
			if defaultValue != nil {
				values[from][to] = *defaultValue
			}
		}
	}
	toSet := map[string]bool{}
	for _, to := range tos {
		toSet[to] = true
	}
	return &TruthTable{
		Froms:      froms,
		Tos:        tos,
		toSet:      toSet,
		Values:     values,
		Bandwidths: bandwidths,
	}
}

// Compare is used to check two truth tables for equality, returning its
// result in the form of a third truth table.  Both tables are expected to
// have identical items.
func (tt *TruthTable) Compare(other *TruthTable) *TruthTable {
	if len(tt.Froms) != len(other.Froms) || len(tt.Tos) != len(other.Tos) {
		fmt.Println("cannot compare tables of different dimensions")
	}
	for i, fr := range tt.Froms {
		if other.Froms[i] != fr {
			fmt.Println(fmt.Printf("cannot compare: from keys at index %d do not match (%s vs %s)", i, other.Froms[i], fr))
		}
	}
	for i, to := range tt.Tos {
		if other.Tos[i] != to {
			fmt.Println(fmt.Printf("cannot compare: to keys at index %d do not match (%s vs %s)", i, other.Tos[i], to))
		}
	}

	values := map[string]map[string]bool{}
	for from, dict := range tt.Values {
		values[from] = map[string]bool{}
		for to, val := range dict {
			values[from][to] = val == other.Values[from][to]
		}
	}
	return &TruthTable{
		Froms:  tt.Froms,
		Tos:    tt.Tos,
		toSet:  tt.toSet,
		Values: values,
	}
}

// IsComplete returns true if there's a value set for every single pair of items, otherwise it returns false.
func (tt *TruthTable) IsComplete() bool {
	for _, from := range tt.Froms {
		for _, to := range tt.Tos {
			if _, ok := tt.Values[from][to]; !ok {
				return false
			}
		}
	}
	return true
}

// Set sets the value for from->to
func (tt *TruthTable) Set(from, to string, value bool) {
	dict, ok := tt.Values[from]
	if !ok {
		fmt.Println(fmt.Printf("from-key %s not found", from))
	}
	if _, ok := tt.toSet[to]; !ok {
		fmt.Println(fmt.Printf("to-key %s not allowed", to))
	}
	dict[to] = value
}

// SetBandwidth sets the bandwidth for from->to
func (tt *TruthTable) SetBandwidth(from, to string, bandwidth *ProbeJobBandwidthResults) {
	dict, ok := tt.Bandwidths[from]
	if !ok {
		fmt.Println(fmt.Printf("from-key %s not found", from))
	}
	if _, ok := tt.toSet[to]; !ok {
		fmt.Println(fmt.Printf("to-key %s not allowed", to))
	}
	dict[to] = bandwidth
}

// Get gets the specified value
func (tt *TruthTable) Get(from, to string) bool {
	dict, ok := tt.Values[from]
	if !ok {
		fmt.Println(fmt.Printf("from-key %s not found", from))
	}
	val, ok := dict[to]
	if !ok {
		fmt.Println(fmt.Printf("to-key %s not found in map (%+v)", to, dict))
	}
	return val
}

// GetBandwidth gets the specified bandwidth for from->to
func (tt *TruthTable) GetBandwidth(from, to string) *ProbeJobBandwidthResults {
	dict, ok := tt.Bandwidths[from]
	if !ok {
		fmt.Println(fmt.Printf("from-key %s not found", from))
	}
	bandwidth, ok := dict[to]
	if !ok {
		fmt.Println(fmt.Printf("to-key %s not found in map (%+v)", to, dict))
	}
	return bandwidth
}

// PrettyPrint produces a nice visual representation.
func (tt *TruthTable) PrettyPrint(indent string) string {
	header := indent + strings.Join(append([]string{"-\t"}, tt.Tos...), "\t")
	lines := []string{header}
	for _, from := range tt.Froms {
		line := []string{from}
		for _, to := range tt.Tos {
			mark := "X"
			val, ok := tt.Values[from][to]
			if !ok {
				mark = "?"
			} else if val {
				mark = "."
			}
			line = append(line, mark+"\t")
		}
		lines = append(lines, indent+strings.Join(line, "\t"))
	}
	return strings.Join(lines, "\n")
}

// PrettyPrintBandwidth produces a nice visual representation for measured bandwidths.
func (tt *TruthTable) PrettyPrintBandwidth(indent string) string {
	header := indent + strings.Join(append([]string{"-\t"}, tt.Tos...), "\t")
	lines := []string{header}
	for _, from := range tt.Froms {
		line := []string{from}
		for _, to := range tt.Tos {
			bandwidth := tt.Bandwidths[from][to]
			var mark string
			if bandwidth == nil {
				mark = "X"
			} else {
				mark = tt.Bandwidths[from][to].PrettyString(true)
			}
			line = append(line, mark+"\t")
		}
		lines = append(lines, indent+strings.Join(line, "\t"))
	}
	return strings.Join(lines, "\n")
}
