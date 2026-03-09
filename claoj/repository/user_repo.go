package repository

import (
	"context"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// GormUserRepo is the GORM implementation of UserRepo.
type GormUserRepo struct {
	db *gorm.DB
}

// NewGormUserRepo creates a new GormUserRepo.
func NewGormUserRepo(database *gorm.DB) *GormUserRepo {
	if database == nil {
		database = db.DB
	}
	return &GormUserRepo{db: database}
}

// WithTx creates a new repository instance using the given transaction.
func (r *GormUserRepo) WithTx(tx *gorm.DB) *GormUserRepo {
	return &GormUserRepo{db: tx}
}

// GetByID retrieves a user by their ID.
func (r *GormUserRepo) GetByID(ctx context.Context, id uint) (*models.AuthUser, error) {
	var user models.AuthUser
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername retrieves a user by their username.
func (r *GormUserRepo) GetByUsername(ctx context.Context, username string) (*models.AuthUser, error) {
	var user models.AuthUser
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by their email address.
func (r *GormUserRepo) GetByEmail(ctx context.Context, email string) (*models.AuthUser, error) {
	var user models.AuthUser
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create creates a new user.
func (r *GormUserRepo) Create(ctx context.Context, user *models.AuthUser) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update updates an existing user.
func (r *GormUserRepo) Update(ctx context.Context, user *models.AuthUser) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete soft-deletes a user by setting IsActive to false.
func (r *GormUserRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.AuthUser{}).Where("id = ?", id).Update("is_active", false).Error
}

// List retrieves a paginated list of users.
func (r *GormUserRepo) List(ctx context.Context, offset, limit int) ([]models.AuthUser, int64, error) {
	var users []models.AuthUser
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.AuthUser{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Search searches for users by username or email.
func (r *GormUserRepo) Search(ctx context.Context, query string, offset, limit int) ([]models.AuthUser, int64, error) {
	var users []models.AuthUser
	var total int64

	searchPattern := "%" + query + "%"

	if err := r.db.WithContext(ctx).Model(&models.AuthUser{}).
		Where("username LIKE ? OR email LIKE ?", searchPattern, searchPattern).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Where("username LIKE ? OR email LIKE ?", searchPattern, searchPattern).
		Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GormProfileRepo is the GORM implementation of ProfileRepo.
type GormProfileRepo struct {
	db *gorm.DB
}

// NewGormProfileRepo creates a new GormProfileRepo.
func NewGormProfileRepo(database *gorm.DB) *GormProfileRepo {
	if database == nil {
		database = db.DB
	}
	return &GormProfileRepo{db: database}
}

// GetByUserID retrieves a profile by user ID.
func (r *GormProfileRepo) GetByUserID(ctx context.Context, userID uint) (*models.Profile, error) {
	var profile models.Profile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetByUsername retrieves a profile by username (joined with user).
func (r *GormProfileRepo) GetByUsername(ctx context.Context, username string) (*models.Profile, error) {
	var profile models.Profile
	if err := r.db.WithContext(ctx).
		Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
		Where("auth_user.username = ?", username).
		First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

// Create creates a new profile.
func (r *GormProfileRepo) Create(ctx context.Context, profile *models.Profile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

// Update updates an existing profile.
func (r *GormProfileRepo) Update(ctx context.Context, profile *models.Profile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

// List retrieves a paginated list of profiles.
func (r *GormProfileRepo) List(ctx context.Context, offset, limit int, excludeUnlisted bool) ([]models.Profile, int64, error) {
	var profiles []models.Profile
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Profile{}).Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id")

	if excludeUnlisted {
		query = query.Where("is_unlisted = ?", false)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("performance_points DESC").Offset(offset).Limit(limit).Find(&profiles).Error; err != nil {
		return nil, 0, err
	}

	return profiles, total, nil
}

// GetTopUsers retrieves top users by performance points.
func (r *GormProfileRepo) GetTopUsers(ctx context.Context, limit int) ([]models.Profile, error) {
	var profiles []models.Profile
	if err := r.db.WithContext(ctx).
		Where("is_unlisted = ?", false).
		Order("performance_points DESC").
		Limit(limit).
		Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}

// UpdateRating updates a user's rating.
func (r *GormProfileRepo) UpdateRating(ctx context.Context, userID uint, rating int) error {
	return r.db.WithContext(ctx).
		Model(&models.Profile{}).
		Where("user_id = ?", userID).
		Update("rating", rating).Error
}
