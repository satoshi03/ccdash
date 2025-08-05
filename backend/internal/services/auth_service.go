package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ccdash-backend/internal/models"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthService struct {
	db             *sql.DB
	jwtSecret      []byte
	tokenDuration  time.Duration
	refreshDuration time.Duration
	auditService   *AuditService
}

// Claims represents JWT claims
type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func NewAuthService(db *sql.DB, jwtSecret string, auditService *AuditService) *AuthService {
	return &AuthService{
		db:              db,
		jwtSecret:       []byte(jwtSecret),
		tokenDuration:   15 * time.Minute,  // Short-lived access tokens
		refreshDuration: 7 * 24 * time.Hour, // 7 days for refresh tokens
		auditService:    auditService,
	}
}

// RegisterUser creates a new user account
func (s *AuthService) RegisterUser(req models.UserRegistrationRequest, ipAddress, userAgent string) (*models.User, error) {
	// Check if user already exists
	existingUser, err := s.GetUserByEmail(req.Email)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		s.auditService.LogEvent(nil, req.Email, "user.register", "users", 
			fmt.Sprintf(`{"email": "%s", "reason": "email_already_exists"}`, req.Email), 
			ipAddress, userAgent, false)
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Set default roles if not provided
	roles := req.Roles
	if len(roles) == 0 {
		roles = []string{"user"} // Default role
	}

	// Validate roles
	for _, role := range roles {
		if _, exists := models.DefaultRoles[role]; !exists {
			return nil, fmt.Errorf("invalid role: %s", role)
		}
	}

	// Serialize roles to JSON
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize roles: %w", err)
	}

	// Create user
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Roles:        roles,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}

	query := `
		INSERT INTO users (id, email, password_hash, roles, created_at, updated_at, is_active, failed_login_attempts)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0)
	`
	_, err = s.db.Exec(query, user.ID, user.Email, user.PasswordHash, string(rolesJSON), 
		user.CreatedAt, user.UpdatedAt, user.IsActive)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Log successful registration
	s.auditService.LogEvent(&user.ID, user.Email, "user.register", "users",
		fmt.Sprintf(`{"email": "%s", "roles": %s}`, user.Email, string(rolesJSON)),
		ipAddress, userAgent, true)

	log.Printf("User registered successfully: %s", user.Email)
	return user, nil
}

// LoginUser authenticates a user and returns tokens
func (s *AuthService) LoginUser(req models.UserLoginRequest, ipAddress, userAgent string) (*models.LoginResponse, error) {
	user, err := s.GetUserByEmail(req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			s.auditService.LogEvent(nil, req.Email, "user.login", "auth",
				fmt.Sprintf(`{"email": "%s", "reason": "user_not_found"}`, req.Email),
				ipAddress, userAgent, false)
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		s.auditService.LogEvent(&user.ID, user.Email, "user.login", "auth",
			fmt.Sprintf(`{"email": "%s", "reason": "account_inactive"}`, req.Email),
			ipAddress, userAgent, false)
		return nil, fmt.Errorf("account is inactive")
	}

	// Check if user is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		s.auditService.LogEvent(&user.ID, user.Email, "user.login", "auth",
			fmt.Sprintf(`{"email": "%s", "reason": "account_locked", "locked_until": "%s"}`, 
				req.Email, user.LockedUntil.Format(time.RFC3339)),
			ipAddress, userAgent, false)
		return nil, fmt.Errorf("account is locked until %s", user.LockedUntil.Format(time.RFC3339))
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		// Increment failed login attempts
		s.incrementFailedLoginAttempts(user.ID)
		
		s.auditService.LogEvent(&user.ID, user.Email, "user.login", "auth",
			fmt.Sprintf(`{"email": "%s", "reason": "invalid_password"}`, req.Email),
			ipAddress, userAgent, false)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset failed login attempts on successful login
	s.resetFailedLoginAttempts(user.ID)

	// Update last login
	s.updateLastLogin(user.ID)

	// Generate tokens
	accessToken, err := s.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Log successful login
	s.auditService.LogEvent(&user.ID, user.Email, "user.login", "auth",
		fmt.Sprintf(`{"email": "%s", "success": true}`, req.Email),
		ipAddress, userAgent, true)

	return &models.LoginResponse{
		User:         *user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.tokenDuration.Seconds()),
	}, nil
}

// RefreshAccessToken generates a new access token using a refresh token
func (s *AuthService) RefreshAccessToken(refreshTokenString string, ipAddress, userAgent string) (*models.LoginResponse, error) {
	// Hash the refresh token to find it in database
	hasher := sha256.New()
	hasher.Write([]byte(refreshTokenString))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	var refreshToken models.RefreshToken
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, is_revoked
		FROM refresh_tokens
		WHERE token_hash = ? AND is_revoked = FALSE AND expires_at > ?
	`
	err := s.db.QueryRow(query, tokenHash, time.Now()).Scan(
		&refreshToken.ID, &refreshToken.UserID, &refreshToken.TokenHash,
		&refreshToken.ExpiresAt, &refreshToken.CreatedAt,
		&refreshToken.RevokedAt, &refreshToken.IsRevoked,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			s.auditService.LogEvent(nil, "", "token.refresh", "auth",
				`{"reason": "invalid_refresh_token"}`, ipAddress, userAgent, false)
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	// Get user
	user, err := s.GetUserByID(refreshToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if !user.IsActive {
		s.auditService.LogEvent(&user.ID, user.Email, "token.refresh", "auth",
			`{"reason": "account_inactive"}`, ipAddress, userAgent, false)
		return nil, fmt.Errorf("account is inactive")
	}

	// Generate new access token
	accessToken, err := s.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token and revoke the old one
	newRefreshToken, err := s.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Revoke old refresh token
	s.RevokeRefreshToken(refreshToken.ID)

	// Log successful token refresh
	s.auditService.LogEvent(&user.ID, user.Email, "token.refresh", "auth",
		fmt.Sprintf(`{"user_id": "%s", "success": true}`, user.ID),
		ipAddress, userAgent, true)

	return &models.LoginResponse{
		User:         *user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.tokenDuration.Seconds()),
	}, nil
}

// GenerateAccessToken creates a new JWT access token
func (s *AuthService) GenerateAccessToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Roles:  user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "ccdash",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// GenerateRefreshToken creates a new refresh token
func (s *AuthService) GenerateRefreshToken(userID string) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash token for storage
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Store in database
	refreshToken := models.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.refreshDuration),
		CreatedAt: time.Now(),
		IsRevoked: false,
	}

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, is_revoked)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query, refreshToken.ID, refreshToken.UserID, refreshToken.TokenHash,
		refreshToken.ExpiresAt, refreshToken.CreatedAt, refreshToken.IsRevoked)
	if err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return token, nil
}

// ValidateAccessToken validates a JWT access token and returns claims
func (s *AuthService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RevokeRefreshToken revokes a refresh token
func (s *AuthService) RevokeRefreshToken(tokenID string) error {
	query := `UPDATE refresh_tokens SET is_revoked = TRUE, revoked_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), tokenID)
	return err
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(id string) (*models.User, error) {
	var user models.User
	var rolesJSON string

	query := `
		SELECT id, email, password_hash, roles, created_at, updated_at, last_login, 
		       is_active, failed_login_attempts, locked_until
		FROM users WHERE id = ?
	`
	err := s.db.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &rolesJSON,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
		&user.IsActive, &user.FailedLoginAttempts, &user.LockedUntil,
	)
	if err != nil {
		return nil, err
	}

	// Deserialize roles
	err = json.Unmarshal([]byte(rolesJSON), &user.Roles)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize roles: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *AuthService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	var rolesJSON string

	query := `
		SELECT id, email, password_hash, roles, created_at, updated_at, last_login,
		       is_active, failed_login_attempts, locked_until
		FROM users WHERE email = ?
	`
	err := s.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &rolesJSON,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
		&user.IsActive, &user.FailedLoginAttempts, &user.LockedUntil,
	)
	if err != nil {
		return nil, err
	}

	// Deserialize roles
	err = json.Unmarshal([]byte(rolesJSON), &user.Roles)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize roles: %w", err)
	}

	return &user, nil
}

// Helper methods

func (s *AuthService) incrementFailedLoginAttempts(userID string) {
	query := `
		UPDATE users 
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE 
		        WHEN failed_login_attempts + 1 >= 5 THEN datetime('now', '+1 hour')
		        ELSE locked_until
		    END
		WHERE id = ?
	`
	s.db.Exec(query, userID)
}

func (s *AuthService) resetFailedLoginAttempts(userID string) {
	query := `UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE id = ?`
	s.db.Exec(query, userID)
}

func (s *AuthService) updateLastLogin(userID string) {
	query := `UPDATE users SET last_login = ? WHERE id = ?`
	s.db.Exec(query, time.Now(), userID)
}

// HasPermission checks if a user has a specific permission
func (s *AuthService) HasPermission(user *models.User, permission models.Permission) bool {
	for _, role := range user.Roles {
		if permissions, exists := models.DefaultRoles[role]; exists {
			for _, p := range permissions {
				if p == permission {
					return true
				}
			}
		}
	}
	return false
}

// HasAnyRole checks if a user has any of the specified roles
func (s *AuthService) HasAnyRole(user *models.User, roles ...string) bool {
	userRolesMap := make(map[string]bool)
	for _, role := range user.Roles {
		userRolesMap[role] = true
	}

	for _, role := range roles {
		if userRolesMap[role] {
			return true
		}
	}
	return false
}

// LogoutUser revokes all refresh tokens for a user
func (s *AuthService) LogoutUser(userID string, ipAddress, userAgent string) error {
	query := `UPDATE refresh_tokens SET is_revoked = TRUE, revoked_at = ? WHERE user_id = ? AND is_revoked = FALSE`
	_, err := s.db.Exec(query, time.Now(), userID)
	
	if err == nil {
		// Get user for audit log
		user, err := s.GetUserByID(userID)
		if err == nil {
			s.auditService.LogEvent(&userID, user.Email, "user.logout", "auth",
				fmt.Sprintf(`{"user_id": "%s"}`, userID), ipAddress, userAgent, true)
		}
	}
	
	return err
}