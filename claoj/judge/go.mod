module github.com/CLAOJ/claoj-go/judge

go 1.24

require (
	github.com/seccomp/libseccomp-golang v0.11.0
	gopkg.in/yaml.v3 v3.0.1
)

// Note: seccomp requires CGO and libseccomp installed
// On Debian/Ubuntu: apt-get install libseccomp-dev
