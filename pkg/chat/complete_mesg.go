package chat

import (
	"app/pkg/util"
	"fmt"
	"time"
)

//Complete message structure
type CompleteMessage struct {
	Buffer         string
	ClientIdentity string
	Timestamp      time.Time
}

//Stringify complete message struct
func (cm *CompleteMessage) String() string {
	return fmt.Sprintf("[%s] %s %s\n",
		util.TimeFormat(cm.Timestamp),
		cm.ClientIdentity,
		cm.Buffer,
	)
}
