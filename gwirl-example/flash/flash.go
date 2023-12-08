package flash

type FlashType uint

const (
	err FlashType = iota
	warning
	success
	info
)

type Flash struct {
	Type    FlashType `json:"type"`
	Message string    `json:"message"`
}
