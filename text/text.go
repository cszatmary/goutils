// Package text provides various utilities for working with text.
// Most APIs have both string and []byte versions.
package text

import (
	"strings"
)

// ExpandVariables replaces ${var} in the byte slice based on the mapping function.
// The returned byte slice is a copy of src with the replacements made, src is not modified.
// If src contains no variables, src is returned as is.
func ExpandVariables(src []byte, mapping func(string) string) []byte {
	var buf []byte
	end := 0
	for i := 0; i < len(src); i++ {
		if i+2 > len(src) {
			// Not enough chars left, can't be a variable
			break
		}
		if !(src[i] == '$' && src[i+1] == '{') {
			continue
		}
		// Lazily initialize buf, explicitly allocate an array to save on allocations
		if buf == nil {
			buf = make([]byte, 0, 2*len(src))
		}
		buf = append(buf, src[end:i]...)

		// Scan until we find a closing brace
		varStart := i + 2
		varEnd := -1
		for j := varStart; j < len(src); j++ {
			if src[j] == '}' {
				varEnd = j
				break
			}
		}
		if varEnd == -1 {
			// Bad syntax `${`, just ignore
			i++
			continue
		}
		if varEnd == varStart {
			// Bad syntax `${}`, just ignore
			i += 2
			continue
		}
		name := src[varStart:varEnd]
		buf = append(buf, mapping(string(name))...)
		i += len(name) + 2
		end = i + 1
	}
	if buf == nil {
		return src
	}
	buf = append(buf, src[end:]...)
	return buf
}

// ExpandVariablesString replaces ${var} in the string based on the mapping function.
func ExpandVariablesString(src string, mapping func(string) string) string {
	var sb *strings.Builder
	end := 0
	for i := 0; i < len(src); i++ {
		if i+2 > len(src) {
			// Not enough chars left, can't be a variable
			break
		}
		if !(src[i] == '$' && src[i+1] == '{') {
			continue
		}
		// Lazily initialize sb, do an explicit grow to save on allocations
		if sb == nil {
			sb = &strings.Builder{}
			sb.Grow(2 * len(src))
		}
		sb.WriteString(src[end:i])

		// Scan until we find a closing brace
		varStart := i + 2
		varEnd := -1
		for j := varStart; j < len(src); j++ {
			if src[j] == '}' {
				varEnd = j
				break
			}
		}
		if varEnd == -1 {
			// Bad syntax `${`, just ignore
			i++
			continue
		}
		if varEnd == varStart {
			// Bad syntax `${}`, just ignore
			i += 2
			continue
		}
		name := src[varStart:varEnd]
		sb.WriteString(mapping(name))
		i += len(name) + 2
		end = i + 1
	}
	if sb == nil {
		return src
	}
	sb.WriteString(src[end:])
	return sb.String()
}

// VariableMapper can be used to expand variables with ExpandVariables or ExpandVariablesString.
// It records any missing variables.
type VariableMapper struct {
	vars       map[string]string
	missingSet map[string]struct{} // missing variables to check existence
	missing    []string            // missing variables in order
}

// NewVariableMapper creates a new VariableMapper that uses vars as the values for expanded variables.
func NewVariableMapper(vars map[string]string) *VariableMapper {
	return &VariableMapper{vars: vars, missingSet: make(map[string]struct{})}
}

// Missing returns all missing variables that were encountered in order.
// A missing variable is only included once, duplicates are removed.
func (vm *VariableMapper) Missing() []string {
	return vm.missing
}

// Map maps a variable name to its value. It can be passed to ExpandVariables or ExpandVariablesString.
func (vm *VariableMapper) Map(name string) string {
	if v, ok := vm.vars[name]; ok {
		return v
	}
	// Check if the same variable was already recorded as missing
	if _, ok := vm.missingSet[name]; !ok {
		// Record as missing
		vm.missingSet[name] = struct{}{}
		vm.missing = append(vm.missing, name)
	}
	return ""
}
