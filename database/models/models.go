package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Client represents a registered client device
type Client struct {
	UUID           string  `json:"uuid,omitempty" gorm:"type:varchar(36);primaryKey"`
	Token          string  `json:"token,omitempty" gorm:"type:varchar(255);unique;not null"`
	Name           string  `json:"name" gorm:"type:varchar(100)"`
	CpuName        string  `json:"cpu_name" gorm:"type:varchar(100)"`
	Virtualization string  `json:"virtualization" gorm:"type:varchar(50)"`
	Arch           string  `json:"arch" gorm:"type:varchar(50)"`
	CpuCores       int     `json:"cpu_cores" gorm:"type:int"`
	OS             string  `json:"os" gorm:"type:varchar(100)"`
	GpuName        string  `json:"gpu_name" gorm:"type:varchar(100)"`
	IPv4           string  `json:"ipv4,omitempty" gorm:"type:varchar(100)"`
	IPv6           string  `json:"ipv6,omitempty" gorm:"type:varchar(100)"`
	Region         string  `json:"region" gorm:"type:varchar(100)"`
	Remark         string  `json:"remark,omitempty" gorm:"type:longtext"`
	PublicRemark   string  `json:"public_remark,omitempty" gorm:"type:longtext"`
	MemTotal       int64   `json:"mem_total" gorm:"type:bigint"`
	SwapTotal      int64   `json:"swap_total" gorm:"type:bigint"`
	DiskTotal      int64   `json:"disk_total" gorm:"type:bigint"`
	Version        string  `json:"version,omitempty" gorm:"type:varchar(100)"`
	Weight         int     `json:"weight" gorm:"type:int"`
	Price          float64 `json:"price"`
	BillingCycle   int     `json:"billing_cycle"`
	AutoRenewal    bool    `json:"auto_renewal" gorm:"default:false"` // 是否自动续费
	Currency       string  `json:"currency" gorm:"type:varchar(20);default:'$'"`
	ExpiredAt      UTCTime `json:"expired_at" gorm:"type:timestamp"`
	Group          string  `json:"group" gorm:"type:varchar(100)"`
	Tags           string  `json:"tags" gorm:"type:text"` // split by ';'
	CreatedAt      UTCTime `json:"created_at"`
	UpdatedAt      UTCTime `json:"updated_at"`
}

// User represents an authenticated user
type User struct {
	UUID      string    `json:"uuid,omitempty" gorm:"type:varchar(36);primaryKey"`
	Username  string    `json:"username" gorm:"type:varchar(50);unique;not null"`
	Passwd    string    `json:"passwd,omitempty" gorm:"type:varchar(255);not null"` // Hashed password
	SSOType   string    `json:"sso_type" gorm:"type:varchar(20)"`                   // e.g., "github", "google"
	SSOID     string    `json:"sso_id" gorm:"type:varchar(100)"`                    // OAuth provider's user ID
	TwoFactor string    `json:"two_factor,omitempty" gorm:"type:varchar(255)"`      // 2FA secret
	Sessions  []Session `json:"sessions,omitempty" gorm:"foreignKey:UUID;references:UUID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
	CreatedAt UTCTime   `json:"created_at"`
	UpdatedAt UTCTime   `json:"updated_at"`
}

// Session manages user sessions
type Session struct {
	UUID            string  `json:"uuid" gorm:"type:varchar(36)"`
	Session         string  `json:"session" gorm:"type:varchar(255);primaryKey;uniqueIndex:idx_sessions_session;not null"`
	UserAgent       string  `json:"user_agent" gorm:"type:text"`
	Ip              string  `json:"ip" gorm:"type:varchar(100)"`
	LoginMethod     string  `json:"login_method" gorm:"type:varchar(50)"`
	LatestOnline    UTCTime `json:"latest_online" gorm:"type:timestamp"`
	LatestUserAgent string  `json:"latest_user_agent" gorm:"type:text"`
	LatestIp        string  `json:"latest_ip" gorm:"type:varchar(100)"`
	Expires         UTCTime `json:"expires" gorm:"not null"`
	CreatedAt       UTCTime `json:"created_at"`
}

// Record logs client metrics over time
type Record struct {
	Client         string  `json:"client" gorm:"type:varchar(36);index"`
	Time           UTCTime `json:"time" gorm:"index"`
	Cpu            float32 `json:"cpu" gorm:"type:decimal(5,2)"` // e.g., 75.50%
	Gpu            float32 `json:"gpu" gorm:"type:decimal(5,2)"`
	Ram            int64   `json:"ram" gorm:"type:bigint"`
	RamTotal       int64   `json:"ram_total" gorm:"type:bigint"`
	Swap           int64   `json:"swap" gorm:"type:bigint"`
	SwapTotal      int64   `json:"swap_total" gorm:"type:bigint"`
	Load           float32 `json:"load" gorm:"type:decimal(5,2)"`
	Temp           float32 `json:"temp" gorm:"type:decimal(5,2)"`
	Disk           int64   `json:"disk" gorm:"type:bigint"`
	DiskTotal      int64   `json:"disk_total" gorm:"type:bigint"`
	NetIn          int64   `json:"net_in" gorm:"type:bigint"`
	NetOut         int64   `json:"net_out" gorm:"type:bigint"`
	NetTotalUp     int64   `json:"net_total_up" gorm:"type:bigint"`
	NetTotalDown   int64   `json:"net_total_down" gorm:"type:bigint"`
	Process        int     `json:"process"`
	Connections    int     `json:"connections"`
	ConnectionsUdp int     `json:"connections_udp"`
}

// StringArray represents a slice of strings stored as JSON in the database
// StringArray 存储为 JSON 的字符串切片类型
type StringArray []string

func (sa *StringArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan StringArray: value is not []byte")
	}
	return json.Unmarshal(bytes, sa)
}

func (sa StringArray) Value() (driver.Value, error) {
	return json.Marshal(sa)
}
