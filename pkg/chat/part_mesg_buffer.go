package chat

import (
	"bytes"
	"time"
)

const (
	MAX_BUFFER_LEN         = 8   //Max buffer len
	LAST_MESG_ADD_DELAY_MS = 100 //last non-empty message in partial message buffer gets older than 100ms since receiving it.
)

//Part Message buffer contains 8 strings
type PartMessageBuffer struct {
	Buffer        []*PartMessage
	fillOutUpdate OnFillOutUpdater
}

//Construct new part mesg buffer
func NewPartMessageBuffer(f OnFillOutUpdater) *PartMessageBuffer {
	//make 0 len and MAX_BUFFER_LEN capacity buffer
	return &PartMessageBuffer{make([]*PartMessage, 0, MAX_BUFFER_LEN), f}
}

//Add new message
func (pmb *PartMessageBuffer) Add(msg, identity string) {
	partMesg := &PartMessage{msg, identity, time.Now()}
	//fmt.Println(fmt.Sprintf("Part Mesg Buf: added new part msg: %#v", partMesg) )
	pmb.Buffer = append(pmb.Buffer, partMesg)
	//Check if buffer is full and we ready to notify fillOutUpdate object with complete message
	pmb.checkBufferIsFull()
}

//Check all mesgs are in their places
func (pmb *PartMessageBuffer) checkBufferIsFull() {
	//Wait for 100ms after last item was added to buffer
	time.Sleep(time.Millisecond * LAST_MESG_ADD_DELAY_MS)
	//fmt.Println(fmt.Sprintf("Part Mesg Buf: check overfill: len of buffer is %d", len(pmb.Buffer)))
	if len(pmb.Buffer) == MAX_BUFFER_LEN {
		//fmt.Println(fmt.Sprintf("Part Mesg Buf: check buffer is full: len is %d, found complete msg and send to server", MAX_BUFFER_LEN))
		pmb.fillOutUpdate.FillOutUpdate(pmb.ToCompleteMesg())
		//Make buffer empty
		pmb.Buffer = pmb.Buffer[:0]
	}
}

//Get all buffered part mesgs together and found new complete message
func (pmb *PartMessageBuffer) ToCompleteMesg() *CompleteMessage {
	var body bytes.Buffer
	firstPm := *pmb.Buffer[0]
	firstPmTimestampStr := firstPm.Timestamp
	authorIdentity := firstPm.ClientIdentity

	for _, v := range pmb.Buffer {
		body.WriteString(v.Data)
	}

	return &CompleteMessage{body.String(), authorIdentity, firstPmTimestampStr}
}
