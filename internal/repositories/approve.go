package repositories

import (
	"errors"
	"fmt"
	"ghost-approve/internal/models"
	"ghost-approve/pkg/botErrors"
	"ghost-approve/pkg/db/postgres"
	"gorm.io/gorm"
	"log"
	"time"
)

func CreateApprove(approve *models.Approval) (*models.Approval, error) {
	if err := postgres.GetDB().Create(approve).Error; err != nil {
		return nil, err
	}
	return approve, nil
}

func FindPendingApprovalsByUserID(userID string) ([]models.Approval, error) {
	var approvals []models.Approval

	query := `
		SELECT a.*
		FROM approvals a
		JOIN approval_users au ON a.id = au.approval_id
		WHERE au.user_id = ? 
		AND a.status = ?
	`
	err := postgres.GetDB().Raw(query, userID, models.StatusPending).Scan(&approvals).Error
	if err != nil {
		return nil, err
	}

	var validApprovals []models.Approval
	for _, approval := range approvals {
		isApproved, err := IsUserApproved(approval.ID, userID)
		if err != nil {
			return nil, err
		}
		isRejected, err := IsUserRejected(approval.ID, userID)
		if err != nil {
			return nil, err
		}

		if !isApproved && !isRejected {
			validApprovals = append(validApprovals, approval)
		}
	}
	return validApprovals, err

}

func GetApproveByID(approveID int) (*models.Approval, error) {
	var approve models.Approval

	err := postgres.GetDB().Preload("Author").
		First(&approve, approveID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("approve not found")
	} else if err != nil {
		return nil, err
	}

	return &approve, nil
}

// AddUserToApprovedUsers добавляет пользователя в список ApproveBy для указанного approveID
func AddUserToApprovedUsers(approveID int, user *models.User) error {
	var approve models.Approval

	if err := postgres.GetDB().First(&approve, approveID).Error; err != nil {

		return err
	}

	isRejected, err := IsUserRejected(approveID, user.ID)
	isApproved, err := IsUserApproved(approveID, user.ID)

	if err != nil {
		return err
	}

	if isRejected || isApproved {
		return botErrors.ErrAlreadyHasResponse
	}
	approvedUser := models.ApprovedUser{
		ApproveID: approveID,
		UserID:    user.ID,
		CreatedAt: time.Now(),
	}
	err = postgres.GetDB().Create(&approvedUser).Error
	return err
}

// AddUserToRejectedBy добавляет пользователя в список ApproveBy для указанного approveID
func AddUserToRejectedBy(approveID int, user *models.User) error {
	var approve models.Approval

	if err := postgres.GetDB().First(&approve, approveID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("approve not found")
		}
		return err
	}

	//
	isRejected, err := IsUserRejected(approveID, user.ID)
	isApproved, err := IsUserApproved(approveID, user.ID)

	if err != nil {
		return err
	}

	if isRejected || isApproved {
		return botErrors.ErrAlreadyHasResponse
	}
	rejectedUser := models.RejectedUser{
		ApproveID: approveID,
		UserID:    user.ID,
		CreatedAt: time.Now(),
	}
	err = postgres.GetDB().Create(&rejectedUser).Error
	return err

}

// IsUserRejected – проверяет, отклонил ли пользователь этот апрув
func IsUserRejected(approveID int, userID string) (bool, error) {
	var exists bool

	err := postgres.GetDB().Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM rejected_users ru
			WHERE ru.approve_id = ? AND ru.user_id = ?
		) AS exists
	`, approveID, userID).Scan(&exists).Error

	if err != nil {
		return false, err
	}

	return exists, nil
}

// IsUserApproved – проверяет, подтвердил ли пользователь этот апрув
func IsUserApproved(approveID int, userID string) (bool, error) {
	var exists bool

	err := postgres.GetDB().Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM approved_users au
			WHERE au.approve_id = ? AND au.user_id = ?
		) AS exists
	`, approveID, userID).Scan(&exists).Error

	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetUsersNotReacted получает всех юзеров, которые не дали отклик
func GetUsersNotReacted(approvalID int) ([]string, error) {
	var notRespondedUsers []string

	// Выполнение запроса с фильтрацией пользователей, которые зарегистрированы
	// и не дали отклик (не находятся в approved_users и rejected_users)
	query := `
	SELECT u.id
	FROM users u
	INNER JOIN approval_users au ON au.user_id = u.id
	WHERE u.registered = true
	  AND au.approval_id = ?
	  AND NOT EXISTS (
	      SELECT 1 FROM approved_users au2 
	      WHERE au2.approve_id = ? AND au2.user_id = u.id
	  )
	  AND NOT EXISTS (
	      SELECT 1 FROM rejected_users ru 
	      WHERE ru.approve_id = ? AND ru.user_id = u.id
	  )
`

	err := postgres.GetDB().Raw(query, approvalID, approvalID, approvalID).Scan(&notRespondedUsers).Error
	if err != nil {
		log.Println("SQL Query Error:", err)
		return nil, err
	}

	return notRespondedUsers, nil
}

// CheckNotRegisteredUsers – возвращает ID всех незарегистрированных пользователей
func CheckNotRegisteredUsers(approvalID int) ([]string, error) {
	var notRegisteredUsers []string

	// Выполнение запроса для получения пользователей, которые не зарегистрированы
	// и привязаны к конкретному approvalID
	err := postgres.GetDB().
		Model(&models.User{}).
		Select("users.id").
		Joins("INNER JOIN approval_users ON approval_users.user_id = users.id").
		Where("users.registered = false").
		Where("approval_users.approval_id = ?", approvalID).
		Scan(&notRegisteredUsers).Error

	if err != nil {
		log.Println("SQL Query Error:", err)
		return nil, err
	}

	return notRegisteredUsers, nil
}

// GetApprovalByID – вернет апрув по его ID
func GetApprovalByID(approvalID int) (*models.Approval, error) {
	var approval models.Approval

	err := postgres.GetDB().
		Where("id = ?", approvalID).
		First(&approval).Error

	if err != nil {
		log.Println("Error fetching approval:", err)
		return nil, err
	}

	return &approval, nil
}
func EditApproveStatus(approvalID int, status models.ApproveStatus) error {
	approval, err := GetApprovalByID(approvalID)
	if err != nil || approval == nil {
		return err
	}

	approval.Status = status

	err = postgres.GetDB().Save(&approval).Error

	return err
}

// SetCompletedAt – поставит в CompletedAt апрува настоящее время
func SetCompletedAt(approveID int) error {
	currentTime := time.Now()

	err := postgres.GetDB().Exec(`
		UPDATE approvals
		SET completed_at = ?
		WHERE id = ?
	`, currentTime, approveID).Error
	if err != nil {
		return fmt.Errorf("error setting CompletedAt for approval with ID %d: %v", approveID, err)
	}

	return nil
}

func ManageApprovals(userID string) ([]models.Approval, error) {
	var approvals []models.Approval

	query := `
		SELECT * 
		FROM approvals 
		WHERE author_id = ? AND status = ?
	`

	if err := postgres.GetDB().Raw(query, userID, models.StatusPending).Scan(&approvals).Error; err != nil {
		return nil, err
	}

	return approvals, nil
}
func GetParticipantsByApprovalID(approveID int) ([]models.User, error) {
	var participants []models.User

	err := postgres.GetDB().Raw(`
		SELECT u.*
		FROM users u
		JOIN approval_users au ON u.id = au.user_id
		WHERE au.approval_id = ?
	`, approveID).Scan(&participants).Error

	if err != nil {
		return nil, err
	}

	return participants, nil
}

func ApprovedUsersID(approveID int) ([]string, error) {
	var userIDs []string

	err := postgres.GetDB().Raw(`
		SELECT user_id 
		FROM approved_users
		WHERE approve_id = ?
	`, approveID).Scan(&userIDs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch approved users: %w", err)
	}

	return userIDs, nil
}

func RejectedUsersID(approveID int) ([]string, error) {
	var userIDs []string

	err := postgres.GetDB().Raw(`
		SELECT user_id 
		FROM rejected_users
		WHERE approve_id = ?
	`, approveID).Scan(&userIDs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch approved users: %w", err)
	}

	return userIDs, nil
}

func ApprovedUsers(approveID int) (*[]models.ApprovedUser, error) {
	var users []models.ApprovedUser

	err := postgres.GetDB().Raw(`
		SELECT au.* 
		FROM approved_users au 
		WHERE au.approve_id = ?
	`, approveID).Scan(&users).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch approved users: %w", err)
	}

	return &users, nil
}

func RejectedUsers(approveID int) (*[]models.RejectedUser, error) {
	var users []models.RejectedUser

	err := postgres.GetDB().Raw(`
		SELECT ru.* 
		FROM rejected_users ru 
		WHERE ru.approve_id = ?
	`, approveID).Scan(&users).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch approved users: %w", err)
	}

	return &users, nil
}

// RegisteredUsersID возвращает ID зарегистрированных пользователей, связанных с апрувом
func RegisteredUsersID(approveID int) ([]string, error) {
	var userIDs []string

	query := `
		SELECT u.id
		FROM users u
		JOIN approval_users au ON au.user_id = u.id
		WHERE au.approval_id = ? AND u.registered = true;
	`

	err := postgres.GetDB().Raw(query, approveID).Scan(&userIDs).Error
	if err != nil {
		return nil, err
	}

	return userIDs, nil
}

func EndApprovalsByUserID(authorID string) (*[]models.Approval, error) {
	var irrelevant []models.Approval

	query := `
		SELECT *
		FROM approvals
		WHERE status IN (?, ?, ?, ?, ?)
		AND author_id = ?;
	`

	err := postgres.GetDB().Raw(query, models.StatusApproved, models.StatusPartiallyApproved, models.StatusRejected, models.StatusCanceled, models.StatusExpired, authorID).Scan(&irrelevant).Error
	if err != nil {
		return nil, err
	}

	if len(irrelevant) == 0 {
		return nil, botErrors.NotFoundApprovalsForReports
	}

	return &irrelevant, nil
}

func DeleteApprovalByID(approvalID int) error {
	return postgres.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM approval_users WHERE approval_id = ?", approvalID).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM file_histories WHERE approve_id = ?", approvalID).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM files WHERE approve_id = ?", approvalID).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM approval_reminders WHERE approval_id = ?", approvalID).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM approvals WHERE id = ?", approvalID).Error; err != nil {
			return err
		}

		return nil
	})
}
