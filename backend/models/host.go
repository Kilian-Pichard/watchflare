package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Host status values.
const (
	StatusPending = "pending"
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusPaused  = "paused"
	StatusExpired = "expired"
)

// Agent command types dispatched via HeartbeatResponse.
const (
	CommandCollectPackages = "collect_packages"
	CommandUpdateAgent     = "update_agent"
)

type Host struct {
	ID        string `gorm:"type:char(36);primarykey" json:"id"`
	AgentID   string `gorm:"unique;not null" json:"agent_id"`
	AgentKey  string `gorm:"not null" json:"-"` // AES-256 key, hidden in JSON

	// Set at creation time (admin)
	DisplayName            string  `gorm:"column:display_name;not null" json:"display_name"`
	ConfiguredIP           *string `json:"configured_ip"`
	PreviousConfiguredIP   *string `json:"previous_configured_ip"`
	AllowAnyIPRegistration bool    `gorm:"default:false" json:"allow_any_ip_registration"`
	IgnoreIPMismatch       bool    `gorm:"default:false" json:"ignore_ip_mismatch"`

	// Registration token
	RegistrationToken *string    `json:"-"` // hashed, NULL after registration
	ExpiresAt         *time.Time `json:"expires_at"` // NULL after successful registration

	// Populated by the agent (NULL until first connection)
	AgentVersion    *string `json:"agent_version"`
	Hostname        *string `json:"hostname"`
	IPAddressV4     *string `json:"ip_address_v4"`
	IPAddressV6     *string `json:"ip_address_v6"`
	Platform        *string `json:"platform"`         // "macOS", "Linux", "Windows" (user-friendly)
	PlatformVersion *string `json:"platform_version"` // "15.6.1", "22.04.3"
	PlatformFamily  *string `json:"platform_family"`  // "darwin", "linux", "windows" (technical)
	Architecture    *string `json:"architecture"`     // "arm64", "amd64"
	Kernel          *string `json:"kernel"`           // "24.6.0", "5.15.0-97-generic"
	EnvironmentType *string `json:"environment_type"` // "physical", "vm", "container", etc.
	Hypervisor      *string `json:"hypervisor"`       // "kvm", "vmware", "virtualbox", "hyperv", "xen", "unknown" (empty if physical)
	ContainerRuntime *string `json:"container_runtime"` // "docker", "lxc", "podman", "kubernetes", "unknown" (empty if not in container)
	LastSeen        *time.Time `json:"last_seen"`

	// Statut du host
	// Valeurs: "pending", "online", "offline", "paused", "expired"
	Status string `gorm:"not null;default:pending;index:idx_hosts_status" json:"status"`

	// Timestamp when agent was reactivated (via UUID reuse)
	// NULL if never reactivated or badge was dismissed
	ReactivatedAt *time.Time `json:"reactivated_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides GORM default pluralization
func (Host) TableName() string {
	return "hosts"
}

// BeforeCreate hook to generate UUID before creating a host
func (h *Host) BeforeCreate(tx *gorm.DB) error {
	if h.ID == "" {
		h.ID = uuid.New().String()
	}
	return nil
}
