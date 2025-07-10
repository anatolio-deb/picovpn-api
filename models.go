package main

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	TelegramUsername string `json:"telegramUsername"`
	TelegramID       int64  `json:"telegramId"`
	// PlanID           uint
	// Plan             Plan

	// ChatID           int64  `json:"chatId"`
	// TelegramUsername string `json:"telegramUsername"`
	// Account          string `json:"account"`
	// Wallet           string `json:"wallet"`
}

type Plan struct {
	gorm.Model
	UserID    uint      `json:"userId"`
	User      User      `json:"user"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Password struct {
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirmation"`
}

func (p Password) IsValid() bool {
	return p.Password != "" && p.Password == p.PasswordConfirm
}

type Daemon struct {
	gorm.Model
	Address string `json:"address"`
	Port    int    `json:"port"`
	CertPEM []byte `json:"certPem"`
	// KeyPem  []byte `json:"keyPem"`
}

type DaemonPublic struct {
	Address string `json:"address"`
}

type PlanUpdate struct {
	Months  int    `json:"months"`
	BOC     string `json:"boc"`
	Comment string `json:"comment"`
}
