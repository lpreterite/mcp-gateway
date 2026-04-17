package gwservice

type ServiceState string

const (
	StateUnknown     ServiceState = "unknown"
	StateValid       ServiceState = "valid"
	StateInvalid     ServiceState = "invalid"
	StatePresent     ServiceState = "present"
	StateMissing     ServiceState = "missing"
	StateLoaded      ServiceState = "loaded"
	StateRunning     ServiceState = "running"
	StateStopped     ServiceState = "stopped"
	StateNotRunning  ServiceState = "not_running"
	StateReachable   ServiceState = "reachable"
	StateUnreachable ServiceState = "unreachable"
)

type RunTargetState string

const (
	RunTargetRunning RunTargetState = "running"
	RunTargetStopped RunTargetState = "stopped"
)

type SuggestedActionCode string

const (
	ActionNone               SuggestedActionCode = "none"
	ActionFixConfig          SuggestedActionCode = "fix_config"
	ActionInstallService     SuggestedActionCode = "install_service"
	ActionStartService       SuggestedActionCode = "start_service"
	ActionReloadRegistration SuggestedActionCode = "reload_registration"
	ActionWaitReady          SuggestedActionCode = "wait_ready"
)

type SuggestedAction struct {
	Code    SuggestedActionCode
	Message string
}

type InstallResult struct {
	ServiceName string
	ConfigPath  string
	InstallPath string
}

type ExitCode int

const (
	ExitOK                 ExitCode = 0
	ExitConfigError        ExitCode = 10
	ExitInstallMissing     ExitCode = 20
	ExitRegistrationError  ExitCode = 30
	ExitRuntimeError       ExitCode = 40
	ExitHealthError        ExitCode = 50
	ExitServiceCommandFail ExitCode = 60
)

type CommandError struct {
	Code    ExitCode
	Message string
}

func (e *CommandError) Error() string {
	return e.Message
}
