package teardown

// Stable issue codes for KindTeardown clijson documents (agents / CI).
const (
	CodePodmanError = "teardown_podman_error"
	CodeListError   = "teardown_list_error"
	CodeNetworkWarn = "teardown_network_warning"
	CodeVolumeError = "teardown_volume_error"
)

// TeardownResult captures observable facts after teardown.Execute for human and JSON renderers.
type TeardownResult struct {
	Project        string
	ContainerNames []string
	Network        string
	NetworkRemoved bool
	NetworkKept    bool
	NetworkWarning string
	VolumeNames    []string
}

// ExitCode is the CLI exit code: 1 when runErr is non-nil, otherwise 0.
// Network removal warnings do not set runErr and therefore yield 0.
func ExitCode(runErr error) int {
	if runErr != nil {
		return 1
	}
	return 0
}
