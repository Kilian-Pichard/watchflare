package models

import "time"

type Server struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	AgentID   string `gorm:"unique;not null" json:"agent_id"`
	AgentKey  string `gorm:"not null" json:"-"` // AES-256 key, hidden in JSON

	// Infos saisies lors de la création (admin)
	Name                   string  `gorm:"not null" json:"name"`
	Type                   string  `gorm:"not null" json:"type"` // physical, vm, docker, lxc
	ConfiguredIP           *string `json:"configured_ip"` // Nullable
	AllowAnyIPRegistration bool    `gorm:"default:false" json:"allow_any_ip_registration"`

	// Token d'enregistrement
	RegistrationToken *string    `json:"-"` // Hashé, NULL après enregistrement
	ExpiresAt         *time.Time `json:"expires_at"` // NULL après enregistrement réussi

	// Infos remontées par l'agent (NULL si pas encore installé)
	Hostname    *string `json:"hostname"`
	IPAddressV4 *string `json:"ip_address_v4"`
	IPAddressV6 *string `json:"ip_address_v6"`
	OS          *string `json:"os"`
	OSVersion   *string `json:"os_version"`
	LastSeen    *time.Time `json:"last_seen"`

	// Statut du serveur
	// Valeurs: "pending", "online", "offline", "expired"
	Status string `gorm:"default:pending" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
