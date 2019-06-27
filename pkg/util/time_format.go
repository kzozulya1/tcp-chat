package util

import (
	"time"
)

// Time formatting with parrent
//$0_padded_24_hour:$0_padded_minute:$0_padded_second
func TimeFormat(t time.Time) string {
	return t.Format("15:04:05") //Part of Magic time format pattern `"Mon Jan _2 15:04:05 2006"`
}
