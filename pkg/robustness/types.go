package robustness

type MitigationType string

type ErrorCategory string

const (
	ErrorCategoryNetwork    ErrorCategory = "network"
	ErrorCategoryDatabase   ErrorCategory = "database"
	ErrorCategoryFilesystem ErrorCategory = "filesystem"
	ErrorCategoryMemory     ErrorCategory = "memory"
	ErrorCategoryCPU        ErrorCategory = "cpu"
	ErrorCategorySecurity   ErrorCategory = "security"
	ErrorCategoryValidation ErrorCategory = "validation"
	ErrorCategoryBusiness   ErrorCategory = "business"
	ErrorCategoryExternal   ErrorCategory = "external"
	ErrorCategoryInternal   ErrorCategory = "internal"
)

type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
	SeverityFatal    ErrorSeverity = "fatal"
)

type CircuitStateEnum string

const (
	CircuitClosed   CircuitStateEnum = "closed"
	CircuitOpen     CircuitStateEnum = "open"
	CircuitHalfOpen CircuitStateEnum = "half_open"
)

type DiagnosticType string

const (
	DiagnosticSystem   DiagnosticType = "system"
	DiagnosticNetwork  DiagnosticType = "network"
	DiagnosticStorage  DiagnosticType = "storage"
	DiagnosticMemory   DiagnosticType = "memory"
	DiagnosticCPU      DiagnosticType = "cpu"
	DiagnosticSecurity DiagnosticType = "security"
)

type EmergencyType string

const (
	EmergencySystemCrash        EmergencyType = "system_crash"
	EmergencyResourceExhaustion EmergencyType = "resource_exhaustion"
	EmergencySecurityBreach     EmergencyType = "security_breach"
	EmergencyNetworkFailure     EmergencyType = "network_failure"
	EmergencyDataLoss           EmergencyType = "data_loss"
	EmergencyServiceOutage      EmergencyType = "service_outage"
)

type FaultType string

const (
	FaultNetwork FaultType = "network"
	FaultDisk    FaultType = "disk"
	FaultMemory  FaultType = "memory"
	FaultCPU     FaultType = "cpu"
	FaultProcess FaultType = "process"
	FaultService FaultType = "service"
)

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthCritical  HealthStatus = "critical"
	HealthUnknown   HealthStatus = "unknown"
)

type DegradationLevel string

const (
	DegradationNone     DegradationLevel = "none"
	DegradationMinimal  DegradationLevel = "minimal"
	DegradationModerate DegradationLevel = "moderate"
	DegradationSevere   DegradationLevel = "severe"
	DegradationCritical DegradationLevel = "critical"
)

type RecoveryType string

const (
	RecoveryAutomatic RecoveryType = "automatic"
	RecoveryManual    RecoveryType = "manual"
	RecoveryForced    RecoveryType = "forced"
)

type RecoveryStatus string

const (
	RecoveryPending   RecoveryStatus = "pending"
	RecoveryRunning   RecoveryStatus = "running"
	RecoveryCompleted RecoveryStatus = "completed"
	RecoveryFailed    RecoveryStatus = "failed"
)

type TestStatus string

const (
	TestPassed  TestStatus = "passed"
	TestFailed  TestStatus = "failed"
	TestSkipped TestStatus = "skipped"
	TestTimeout TestStatus = "timeout"
)

type AlertSeverity string

const (
	AlertLow      AlertSeverity = "low"
	AlertMedium   AlertSeverity = "medium"
	AlertHigh     AlertSeverity = "high"
	AlertCritical AlertSeverity = "critical"
)

type ServiceType string

const (
	ServiceCore      ServiceType = "core"
	ServiceSecondary ServiceType = "secondary"
	ServiceAuxiliary ServiceType = "auxiliary"
	ServiceDebug     ServiceType = "debug"
)

type RepairType string

const (
	RepairRestart     RepairType = "restart"
	RepairReconfigure RepairType = "reconfigure"
	RepairReplace     RepairType = "replace"
	RepairCleanup     RepairType = "cleanup"
	RepairUpdate      RepairType = "update"
)

type EmergencyStatus string

const (
	EmergencyDetected   EmergencyStatus = "detected"
	EmergencyResponding EmergencyStatus = "responding"
	EmergencyResolved   EmergencyStatus = "resolved"
	EmergencyFailed     EmergencyStatus = "failed"
)

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepExecuting StepStatus = "executing"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
	StepSkipped   StepStatus = "skipped"
)

type ResourceLimits struct {
	MaxMemoryMB   int64   `yaml:"max_memory_mb"`
	MaxCPUPercent float64 `yaml:"max_cpu_percent"`
	MaxDiskGB     int64   `yaml:"max_disk_gb"`
}
