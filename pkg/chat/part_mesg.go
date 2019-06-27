package chat

import (
	"time"
)

//Part message struct
type PartMessage struct {
	Data           string
	ClientIdentity string
	Timestamp      time.Time
}
