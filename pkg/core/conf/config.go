package conf

// Version is the current release version.
// The default value is used when the binary is built without ldflags injection.
// Production builds should set this via:
//
//	go build -ldflags "-X github.com/boy-hack/ksubdomain/v2/pkg/core/conf.Version=v2.x.y" \
//	    ./cmd/ksubdomain
var Version = "2.4-dev"

const (
	AppName     = "KSubdomain"
	Description = "无状态子域名爆破工具"
)
