package pack

import (
	"strconv"
	"strings"
)

type packerInfo struct {
	// Maximum size of the data, works differently based on type
	// (Does not account for kind/length bytes)
	//
	// string: max encoded bytes
	// slice/map: max elements
	maxSize uint64

	// Ignore this field
	ignore bool

	// Must mark the encoded data with it's kind
	//
	// Necessary for decoding data typed as interface{}
	markType bool

	objects string

	forceAsObject bool

	seen seen
}

type seen []uintptr

func (s *seen) push(a uintptr) error {
	for i := len(*s); i > 0; i-- {
		if (*s)[i-1] == a {
			return ErrCycle
		}
	}

	*s = append(*s, a)

	return nil
}

func (s *seen) pop() {
	*s = (*s)[:len(*s)-1]
}

func parsePackerInfo(tag string, old ...packerInfo) packerInfo {
	var info packerInfo

	if len(old) > 0 {
		info.seen = old[0].seen
	}

	if tag == "" {
		return info
	}

	for _, part := range strings.Split(tag, ";") {

		var (
			key = part
			val = ""
		)

		if idx := strings.IndexByte(key, ':'); idx != -1 {
			key = part[:idx]
			val = part[idx+1:]
		}

		switch key {
		case "max":
			info.maxSize, _ = strconv.ParseUint(val, 10, 64)

		case "ignore":
			info.ignore = true

		case "objects":
			info.objects = val
		}

	}

	return info
}
