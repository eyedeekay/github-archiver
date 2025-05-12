package util

import "log"

var FORCE_PROCESSING = false

func ForceProcessing(e error) bool {
	if e != nil {
		log.Printf("ERROR CONTINUANCE: %v", e)
	}
	if FORCE_PROCESSING {
		return false
	}
	return e != nil
}
