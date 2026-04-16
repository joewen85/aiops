package service

import (
	"errors"

	"gorm.io/gorm"

	"devops-system/backend/internal/auth"
	"devops-system/backend/internal/models"
)

func SeedDefaultAdmin(database *gorm.DB) error {
	if database == nil {
		return nil
	}

	var role models.Role
	err := database.Where("name = ?", "admin").First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role = models.Role{Name: "admin", Description: "built-in administrator", BuiltIn: true}
		if err = database.Create(&role).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	var user models.User
	err = database.Where("username = ?", "admin").First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, hashErr := auth.HashPassword("Admin@123")
		if hashErr != nil {
			return hashErr
		}
		user = models.User{
			Username:     "admin",
			PasswordHash: hash,
			DisplayName:  "System Admin",
			IsActive:     true,
		}
		if err = database.Create(&user).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	var userRole models.UserRole
	err = database.Where("user_id = ? AND role_id = ?", user.ID, role.ID).First(&userRole).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		userRole = models.UserRole{UserID: user.ID, RoleID: role.ID}
		if err = database.Create(&userRole).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}
