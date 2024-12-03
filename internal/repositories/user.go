package repositories

import (
	"errors"
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/pkg/botErrors"
	"ghost-approve/pkg/db/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func init() {
	db = postgres.GetDB()
}
func GetUserByID(id string) *models.User {
	var user models.User
	result := postgres.GetDB().Where("id = ?", id).First(&user)

	//short-circuiting
	if result == nil || result.Error != nil {
		return nil
	}

	return &user
}

func CreateUser(id, firstName, lastName string) models.User {
	user := models.User{
		ID:         id,
		FirstName:  firstName,
		LastName:   lastName,
		Registered: true,
	}
	postgres.GetDB().Create(&user)
	return user
}

func ActivateUser(user *models.User, firstName, lastName string) error {
	if user == nil {
		return errors.New("Не найден пользователь")
	}

	user.FirstName = firstName
	user.LastName = lastName
	user.Registered = true

	result := postgres.GetDB().Save(user)

	if result == nil || result.Error != nil {
		return errors.New("Не удалось сохранить пользователя")
	}

	return nil
}

func CheckEmailsInDB(emails []string) ([]string, []string, error) {
	var foundUsers []models.User

	if err := postgres.GetDB().Where("id IN ? AND registered = ?", emails, true).Find(&foundUsers).Error; err != nil {
		return nil, nil, err
	}

	foundEmailsMap := make(map[string]bool)
	for _, user := range foundUsers {
		foundEmailsMap[user.ID] = true
	}

	var foundEmails []string
	var notFoundEmails []string
	for _, email := range emails {
		if foundEmailsMap[email] {
			foundEmails = append(foundEmails, email)
		} else {
			notFoundEmails = append(notFoundEmails, email)
		}
	}

	return foundEmails, notFoundEmails, nil
}

func GetOrCreateUsersByID(ids []string) ([]models.User, error) {
	var users []models.User

	if err := postgres.GetDB().Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}

	existingIDs := make(map[string]bool)
	for _, user := range users {
		existingIDs[user.ID] = true
	}

	for _, id := range ids {
		if !existingIDs[id] {
			newUser := models.User{ID: id, Registered: false}
			if err := postgres.GetDB().Create(&newUser).Error; err != nil {
				return nil, err
			}
			users = append(users, newUser)
		}
	}

	return users, nil
}

func ApprovalParticipants(approveID int) ([]string, error) {
	var userIDs []string

	if err := postgres.GetDB().Raw(`
			SELECT au.user_id 
			FROM approval_users au
			JOIN approvals a ON a.id = au.approval_id
			WHERE a.id = ?`, approveID).Scan(&userIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to find all participants: %w", err)
	}

	if len(userIDs) == 0 {
		return nil, botErrors.NotFoundUsers
	}

	return userIDs, nil
}
