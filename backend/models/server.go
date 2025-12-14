package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Server struct {
	ID        string `gorm:"type:char(36);primarykey" json:"id"`
	AgentID   string `gorm:"unique;not null" json:"agent_id"`
	AgentKey  string `gorm:"not null" json:"-"` // AES-256 key, hidden in JSON

	// Infos saisies lors de la création (admin)
	Name                   string  `gorm:"not null" json:"name"`
	ConfiguredIP           *string `json:"configured_ip"`           // Nullable
	PreviousConfiguredIP   *string `json:"previous_configured_ip"` // Ancienne IP configurée
	AllowAnyIPRegistration bool    `gorm:"default:false" json:"allow_any_ip_registration"`
	IgnoreIPMismatch       bool    `gorm:"default:false" json:"ignore_ip_mismatch"` // L'utilisateur a choisi d'ignorer l'alerte

	// Token d'enregistrement
	RegistrationToken *string    `json:"-"` // Hashé, NULL après enregistrement
	ExpiresAt         *time.Time `json:"expires_at"` // NULL après enregistrement réussi

	// Infos remontées par l'agent (NULL si pas encore installé)
	Hostname        *string `json:"hostname"`
	IPAddressV4     *string `json:"ip_address_v4"`
	IPAddressV6     *string `json:"ip_address_v6"`
	Platform        *string `json:"platform"`         // "macOS", "Linux", "Windows" (user-friendly)
	PlatformVersion *string `json:"platform_version"` // "15.6.1", "22.04.3"
	PlatformFamily  *string `json:"platform_family"`  // "darwin", "linux", "windows" (technical)
	Architecture    *string `json:"architecture"`     // "arm64", "amd64"
	Kernel          *string `json:"kernel"`           // "24.6.0", "5.15.0-97-generic"
	LastSeen        *time.Time `json:"last_seen"`

	// Statut du serveur
	// Valeurs: "pending", "online", "offline", "expired"
	Status string `gorm:"default:pending" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID before creating a server
func (s *Server) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}
