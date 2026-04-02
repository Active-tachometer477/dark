package dark

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const sourceMappingPrefix = "//# sourceMappingURL=data:application/json;base64,"

// sourceMap represents a parsed source map (v3).
type sourceMap struct {
	Sources      []string `json:"sources"`
	Mappings     string   `json:"mappings"`
	mappingLines [][]mapping
}

type mapping struct {
	genCol     int
	sourceIdx  int
	sourceLine int
	sourceCol  int
}

type originalPos struct {
	source string
	line   int // 1-based
	col    int // 0-based
}

// parseInlineSourceMap extracts and parses an inline source map from bundled JS.
func parseInlineSourceMap(js string) (*sourceMap, error) {
	idx := strings.LastIndex(js, sourceMappingPrefix)
	if idx < 0 {
		return nil, fmt.Errorf("no inline source map found")
	}
	b64 := strings.TrimRight(js[idx+len(sourceMappingPrefix):], " \t\r\n")
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	var sm sourceMap
	if err := json.Unmarshal(data, &sm); err != nil {
		return nil, fmt.Errorf("parse source map JSON: %w", err)
	}
	sm.mappingLines = decodeMappings(sm.Mappings)
	sm.Mappings = "" // free raw string after decoding
	return &sm, nil
}

// stripInlineSourceMap removes the inline source map comment from bundled JS.
func stripInlineSourceMap(js string) string {
	if idx := strings.LastIndex(js, sourceMappingPrefix); idx >= 0 {
		return strings.TrimRight(js[:idx], " \t\r\n")
	}
	return js
}

// lookup finds the original position for a generated line (1-based) and column (0-based).
func (sm *sourceMap) lookup(genLine, genCol int) (originalPos, bool) {
	idx := genLine - 1
	if idx < 0 || idx >= len(sm.mappingLines) {
		return originalPos{}, false
	}
	segs := sm.mappingLines[idx]
	if len(segs) == 0 {
		return originalPos{}, false
	}

	best := segs[0]
	for _, seg := range segs {
		if seg.genCol <= genCol {
			best = seg
		} else {
			break
		}
	}

	source := ""
	if best.sourceIdx >= 0 && best.sourceIdx < len(sm.Sources) {
		source = sm.Sources[best.sourceIdx]
	}
	return originalPos{
		source: source,
		line:   best.sourceLine + 1,
		col:    best.sourceCol,
	}, true
}

// decodeMappings decodes the VLQ-encoded "mappings" string.
func decodeMappings(mappings string) [][]mapping {
	lines := strings.Split(mappings, ";")
	result := make([][]mapping, len(lines))

	var srcIdx, srcLine, srcCol int

	for i, line := range lines {
		if line == "" {
			continue
		}
		segments := strings.Split(line, ",")
		var genCol int
		for _, seg := range segments {
			values := decodeVLQ(seg)
			if len(values) < 4 {
				continue
			}
			genCol += values[0]
			srcIdx += values[1]
			srcLine += values[2]
			srcCol += values[3]
			result[i] = append(result[i], mapping{
				genCol:     genCol,
				sourceIdx:  srcIdx,
				sourceLine: srcLine,
				sourceCol:  srcCol,
			})
		}
	}
	return result
}

// decodeVLQ decodes a single VLQ-encoded segment into a slice of integers.
func decodeVLQ(s string) []int {
	var result []int
	var value int
	var shift uint
	for _, c := range s {
		if c > 127 {
			continue
		}
		digit := vlqCharToInt(byte(c))
		if digit < 0 {
			continue
		}
		value |= (digit & 0x1f) << shift
		shift += 5
		if shift > 60 {
			return nil // overflow protection
		}
		if digit&0x20 == 0 {
			if value&1 != 0 {
				result = append(result, -(value >> 1))
			} else {
				result = append(result, value>>1)
			}
			value = 0
			shift = 0
		}
	}
	return result
}

func vlqCharToInt(c byte) int {
	if c >= 'A' && c <= 'Z' {
		return int(c - 'A')
	}
	if c >= 'a' && c <= 'z' {
		return int(c-'a') + 26
	}
	if c >= '0' && c <= '9' {
		return int(c-'0') + 52
	}
	if c == '+' {
		return 62
	}
	if c == '/' {
		return 63
	}
	return -1
}

// jsStackLineRe matches stack trace line:col at end of line.
var jsStackLineRe = regexp.MustCompile(`(?m)(?:at |@)\S+?:(\d+):(\d+)\s*$`)

// mapErrorWithSourceMap rewrites line numbers in a JS error message using a source map.
func mapErrorWithSourceMap(errMsg string, sm *sourceMap) string {
	if sm == nil {
		return errMsg
	}
	return jsStackLineRe.ReplaceAllStringFunc(errMsg, func(match string) string {
		sub := jsStackLineRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		line, _ := strconv.Atoi(sub[1])
		col, _ := strconv.Atoi(sub[2])
		pos, ok := sm.lookup(line, col)
		if !ok || pos.source == "" {
			return match
		}
		return fmt.Sprintf("at %s:%d:%d", pos.source, pos.line, pos.col)
	})
}
