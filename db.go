package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	dsn := fmt.Sprintf("host=db user=postgres password=%s dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Shanghai", os.Getenv("POSTGRES_PASSWORD"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Error(err)
	}
	err = db.AutoMigrate(&User{})
	if err != nil {
		logrus.Error(err)
	}
	err = db.AutoMigrate(&Plan{})
	if err != nil {
		logrus.Error(err)
	}
	err = db.AutoMigrate(&Daemon{})
	if err != nil {
		logrus.Error(err)
	}
	DB = db
}

func UserGetByTelegramID(id int64) (*User, error) {
	var user *User
	result := DB.First(&user, "telegram_id = ?", id)
	return user, result.Error
}

func PlansGetExpired() ([]Plan, error) {
	plans := make([]Plan, 0)
	result := DB.Where("expires_at <= ?", time.Now()).Find(&plans)
	return plans, result.Error
}

func DaemonsGetAll() ([]Daemon, error) {
	daemons := make([]Daemon, 0)
	result := DB.Find(&daemons)
	if result.Error != nil {
		return nil, result.Error
	}
	return daemons, nil
}

func DaemonGetByAddress(address string) (*Daemon, error) {
	var daemon *Daemon
	result := DB.First(&daemon, "address = ?", address)
	if result.Error != nil {
		return nil, result.Error
	}
	return daemon, nil
}

func PlansGetByTelegramUserID(id int64) (*Plan, error) {
	user, err := UserGetByTelegramID(id)
	if err != nil {
		return nil, err
	}
	plan := Plan{}
	result := DB.First(&plan, "user_id=?", user.ID)
	plan.User = *user
	return &plan, result.Error
}
