package dtos

// InstanceConfigDto carries facts a client cannot work out for itself because
// they depend on how the instance was reached rather than how it was built.
type InstanceConfigDto struct {
	// MaxRequestBytes is the largest request body this origin accepts. Zero means
	// no limit, so an upload need not be split at all.
	MaxRequestBytes int64 `json:"maxRequestBytes" validate:"required"`
}
