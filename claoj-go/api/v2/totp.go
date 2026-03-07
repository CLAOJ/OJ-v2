package v2

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// TOTPKey holds TOTP key information
type TOTPKey struct {
	Secret   string `json:"secret"`
	URL      string `json:"url"`
	QRCode   string `json:"qr_code"` // base64 encoded
}

// encryptSecret encrypts the TOTP secret using AES-GCM
func encryptSecret(plaintext string) (string, error) {
	key := sha256.Sum256([]byte(config.C.App.SecretKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptSecret decrypts the TOTP secret
func decryptSecret(ciphertext string) (string, error) {
	key := sha256.Sum256([]byte(config.C.App.SecretKey))
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// generateQRCodeURL generates a QR code URL for the TOTP secret
func generateQRCodeURL(username, secret string) string {
	issuer := config.C.Email.FromName
	if issuer == "" {
		issuer = "CLAOJ"
	}
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		issuer, username, secret, issuer)
}

// generateQRCodeDataURI generates a base64 data URI for a QR code
func generateQRCodeDataURI(url string) (string, error) {
	// For now, return the URL - frontend can generate the QR code
	// In production, you might want to use a library like skip2/go-qrcode
	return url, nil
}

// TotpSetupRequest - POST /api/v2/auth/totp/setup
// Generates TOTP secret and QR code for setup
type TotpSetupRequest struct {
	Password string `json:"password" binding:"required"`
}

func TotpSetup(c *gin.Context) {
	userID := c.GetUint("userID")
	username := c.GetString("username")

	var req TotpSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Verify password
	var user models.AuthUser
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	match, err := auth.CheckPassword(req.Password, user.Password)
	if err != nil || !match {
		c.JSON(http.StatusUnauthorized, apiError("invalid password"))
		return
	}

	// Check if TOTP is already set up
	var existingDevice models.TotpDevice
	if err := db.DB.Where("user_id = ?", userID).First(&existingDevice).Error; err == nil {
		if existingDevice.Confirmed {
			c.JSON(http.StatusBadRequest, apiError("TOTP is already enabled. Disable it first to reconfigure."))
			return
		}
		// Delete unconfirmed device
		db.DB.Delete(&existingDevice)
	}

	// Generate new TOTP secret
	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      config.C.Email.FromName,
		AccountName: username,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to generate TOTP secret"))
		return
	}

	// Encrypt secret for storage
	encryptedSecret, err := encryptSecret(secret.Secret())
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to encrypt secret"))
		return
	}

	// Store unconfirmed device
	device := models.TotpDevice{
		UserID:    userID,
		Secret:    encryptedSecret,
		Confirmed: false,
		CreatedAt: time.Now(),
	}
	if err := db.DB.Create(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to store TOTP device"))
		return
	}

	// Generate QR code URL
	url := generateQRCodeURL(username, secret.Secret())

	c.JSON(http.StatusOK, gin.H{
		"secret":    secret.Secret(),
		"url":       url,
		"qr_code":   url, // Frontend can generate QR from this URL
		"backup_codes_remaining": 0,
	})
}

// TotpConfirmRequest - POST /api/v2/auth/totp/confirm
// Confirms TOTP setup with verification code
type TotpConfirmRequest struct {
	Code string `json:"code" binding:"required"`
}

func TotpConfirm(c *gin.Context) {
	userID := c.GetUint("userID")

	var req TotpConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Get unconfirmed device
	var device models.TotpDevice
	if err := db.DB.Where("user_id = ? AND confirmed = ?", userID, false).First(&device).Error; err != nil {
		c.JSON(http.StatusBadRequest, apiError("no pending TOTP setup found"))
		return
	}

	// Decrypt secret
	secret, err := decryptSecret(device.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to decrypt secret"))
		return
	}

	// Verify code
	valid := totp.Validate(req.Code, secret)
	if !valid {
		c.JSON(http.StatusUnauthorized, apiError("invalid TOTP code"))
		return
	}

	// Confirm device
	device.Confirmed = true
	if err := db.DB.Save(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to confirm TOTP"))
		return
	}

	// Generate backup codes
	backupCodes, err := generateBackupCodes(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to generate backup codes"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "TOTP enabled successfully",
		"backup_codes": backupCodes,
	})
}

// TotpDisableRequest - POST /api/v2/auth/totp/disable
// Disables TOTP
type TotpDisableRequest struct {
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func TotpDisable(c *gin.Context) {
	userID := c.GetUint("userID")

	var req TotpDisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Verify password
	var user models.AuthUser
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	match, err := auth.CheckPassword(req.Password, user.Password)
	if err != nil || !match {
		c.JSON(http.StatusUnauthorized, apiError("invalid password"))
		return
	}

	// Get confirmed device
	var device models.TotpDevice
	if err := db.DB.Where("user_id = ? AND confirmed = ?", userID, true).First(&device).Error; err != nil {
		c.JSON(http.StatusBadRequest, apiError("TOTP is not enabled"))
		return
	}

	// Decrypt secret and verify code
	secret, err := decryptSecret(device.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to decrypt secret"))
		return
	}

	valid := totp.Validate(req.Code, secret)
	if !valid {
		c.JSON(http.StatusUnauthorized, apiError("invalid TOTP code"))
		return
	}

	// Delete device and backup codes
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.BackupCode{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&device).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to disable TOTP"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "TOTP disabled successfully"})
}

// TotpVerifyRequest - POST /api/v2/auth/totp/verify
// Verifies TOTP code during login
type TotpVerifyRequest struct {
	Username string `json:"username" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

func TotpVerify(c *gin.Context) {
	var req TotpVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Find user
	var user models.AuthUser
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, apiError("invalid credentials"))
		return
	}

	// Get TOTP device
	var device models.TotpDevice
	if err := db.DB.Where("user_id = ? AND confirmed = ?", user.ID, true).First(&device).Error; err != nil {
		c.JSON(http.StatusBadRequest, apiError("TOTP not enabled for this account"))
		return
	}

	// Decrypt secret
	secret, err := decryptSecret(device.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to decrypt secret"))
		return
	}

	// Verify TOTP code
	valid := totp.Validate(req.Code, secret)
	if !valid {
		c.JSON(http.StatusUnauthorized, apiError("invalid TOTP code"))
		return
	}

	// Generate tokens (TOTP verification doesn't have remember_me, use default 7 days)
	accessToken, refreshToken, _, err := auth.GenerateTokens(user.ID, user.Username, user.IsSuperuser, "", false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to generate tokens"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsSuperuser,
		},
	})
}

// TotpStatus - GET /api/v2/auth/totp/status
// Returns TOTP status for current user
func TotpStatus(c *gin.Context) {
	userID := c.GetUint("userID")

	var device models.TotpDevice
	err := db.DB.Where("user_id = ? AND confirmed = ?", userID, true).First(&device).Error

	var backupCount int64
	db.DB.Model(&models.BackupCode{}).Where("user_id = ? AND used = ?", userID, false).Count(&backupCount)

	c.JSON(http.StatusOK, gin.H{
		"enabled":                err == nil,
		"backup_codes_remaining": backupCount,
	})
}

// TotpBackupCodesGenerate - POST /api/v2/auth/totp/backup-codes
// Generates new backup codes
type TotpBackupCodesGenerateRequest struct {
	Password string `json:"password" binding:"required"`
}

func TotpBackupCodesGenerate(c *gin.Context) {
	userID := c.GetUint("userID")

	var req TotpBackupCodesGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Verify password
	var user models.AuthUser
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	match, err := auth.CheckPassword(req.Password, user.Password)
	if err != nil || !match {
		c.JSON(http.StatusUnauthorized, apiError("invalid password"))
		return
	}

	// Check if TOTP is enabled
	var device models.TotpDevice
	if err := db.DB.Where("user_id = ? AND confirmed = ?", userID, true).First(&device).Error; err != nil {
		c.JSON(http.StatusBadRequest, apiError("TOTP is not enabled"))
		return
	}

	// Generate new backup codes
	backupCodes, err := generateBackupCodes(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to generate backup codes"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Backup codes generated successfully",
		"backup_codes": backupCodes,
	})
}

// TotpBackupVerifyRequest - POST /api/v2/auth/totp/verify-backup
// Verifies and uses a backup code
type TotpBackupVerifyRequest struct {
	Username string `json:"username" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

func TotpBackupVerify(c *gin.Context) {
	var req TotpBackupVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Find user
	var user models.AuthUser
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, apiError("invalid credentials"))
		return
	}

	// Hash the provided code
	hash := sha256.Sum256([]byte(req.Code))
	codeHash := hex.EncodeToString(hash[:])

	// Find unused backup code
	var backupCode models.BackupCode
	if err := db.DB.Where("user_id = ? AND code = ? AND used = ?", user.ID, codeHash, false).First(&backupCode).Error; err != nil {
		c.JSON(http.StatusUnauthorized, apiError("invalid backup code"))
		return
	}

	// Mark code as used
	backupCode.Used = true
	if err := db.DB.Save(&backupCode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to use backup code"))
		return
	}

	// Generate tokens (backup code verification doesn't have remember_me, use default 7 days)
	accessToken, refreshToken, _, err := auth.GenerateTokens(user.ID, user.Username, user.IsSuperuser, "", false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to generate tokens"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsSuperuser,
		},
		"warning": "You have used a backup code. Please generate new ones if needed.",
	})
}

// generateBackupCodes generates 10 backup codes for a user
func generateBackupCodes(userID uint) ([]string, error) {
	// Delete existing backup codes
	if err := db.DB.Where("user_id = ?", userID).Delete(&models.BackupCode{}).Error; err != nil {
		return nil, err
	}

	codes := make([]string, 10)
	now := time.Now()

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < 10; i++ {
			// Generate random 8-digit code
			codeBytes := make([]byte, 4)
			if _, err := rand.Read(codeBytes); err != nil {
				return err
			}
			code := fmt.Sprintf("%08d", int(codeBytes[0])<<24|int(codeBytes[1])<<16|int(codeBytes[2])<<8|int(codeBytes[3]))
			code = code[:8]

			// Format as XXXX-XXXX
			formattedCode := code[:4] + "-" + code[4:]
			codes[i] = formattedCode

			// Hash the code for storage
			hash := sha256.Sum256([]byte(formattedCode))
			codeHash := hex.EncodeToString(hash[:])

			backupCode := models.BackupCode{
				UserID:    userID,
				Code:      codeHash,
				Used:      false,
				CreatedAt: now,
			}
			if err := tx.Create(&backupCode).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return codes, nil
}

// CheckTOTPRequired checks if user has TOTP enabled
func CheckTOTPRequired(userID uint) bool {
	var device models.TotpDevice
	err := db.DB.Where("user_id = ? AND confirmed = ?", userID, true).First(&device).Error
	return err == nil
}

// GenerateTempToken generates a temporary token for TOTP challenge
func GenerateTempToken(userID uint, username string) (string, error) {
	// Create a simple temporary token that expires in 5 minutes
	// This token can only be used for TOTP verification
	tokenData := fmt.Sprintf("%d:%s:%d", userID, username, time.Now().Unix())
	hash := sha256.Sum256([]byte(tokenData + config.C.App.SecretKey))
	return hex.EncodeToString(hash[:]), nil
}

// VerifyTempToken verifies a temporary token
func VerifyTempToken(token string, userID uint, username string) bool {
	// For simplicity, we check tokens generated in the last 5 minutes
	now := time.Now().Unix()
	for i := int64(0); i <= 300; i++ { // Check last 5 minutes
		tokenData := fmt.Sprintf("%d:%s:%d", userID, username, now-i)
		hash := sha256.Sum256([]byte(tokenData + config.C.App.SecretKey))
		expectedToken := hex.EncodeToString(hash[:])
		if token == expectedToken {
			return true
		}
	}
	return false
}

// HashBackupCode hashes a backup code using bcrypt for secure storage
func HashBackupCode(code string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(strings.ToUpper(code)), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyBackupCode verifies a backup code against a hash
func VerifyBackupCode(code, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(strings.ToUpper(code)))
	return err == nil
}
