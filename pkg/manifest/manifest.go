package manifest

import (
	"portaptable/pkg/packageinfo"
	"time"
)

type Manifest struct {
	CreatedAt    time.Time                 `json:"created_at"`
	Architecture string                    `json:"architecture"`
	Distribution string                    `json:"distribution"`
	Packages     []packageinfo.PackageInfo `json:"packages"`
}
