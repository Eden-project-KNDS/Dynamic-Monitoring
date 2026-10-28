package monitor

// GPU is the state of a single GPU card
type GPUStat struct {
	Index       int
	UUID        string
	UtilPercent uint32 // 0-100
	JobID       string // "" = card not allocated to any job
}
