package chat

//Interface for notification Compete Message is ready
type OnFillOutUpdater interface {
	FillOutUpdate(*CompleteMessage)
}
