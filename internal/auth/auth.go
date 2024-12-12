package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entUser "github.com/zibbp/ganymede/ent/user"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/user"
	"github.com/zibbp/ganymede/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

type Service struct {
	Store *database.Database
	OAuth struct {
		Provider *oidc.Provider
		Config   oauth2.Config
	}
	EnvConfig *config.EnvConfig
}

func NewService(store *database.Database, envConfig *config.EnvConfig) *Service {
	ctx := context.Background()
	env := config.GetEnvConfig()

	if env.OAuthEnabled {
		// Fetch environment variables
		providerURL := env.OAuthProviderURL
		oauthClientID := env.OAuthClientID
		oauthClientSecret := env.OAuthClientSecret
		oauthRedirectURL := env.OAuthRedirectURL
		if providerURL == "" || oauthClientID == "" || oauthClientSecret == "" || oauthRedirectURL == "" {
			log.Fatal().Msg("missing environment variables for oauth authentication")
		}
		provider, err := oidc.NewProvider(context.Background(), providerURL)
		if err != nil {
			log.Fatal().Err(err).Msg("error creating oauth provider")
		}

		config := oauth2.Config{
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
			RedirectURL:  oauthRedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes: []string{
				oidc.ScopeOpenID,
				"profile",
				"groups",
			},
		}

		err = FetchJWKS(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("error fetching jwks")
		}

		return &Service{
			Store: store,
			OAuth: struct {
				Provider *oidc.Provider
				Config   oauth2.Config
			}{
				Provider: provider,
				Config:   config,
			},
			EnvConfig: envConfig,
		}
	} else {
		return &Service{Store: store}
	}
}

func (s *Service) Register(ctx context.Context, user user.User) (*ent.User, error) {
	if !config.Get().RegistrationEnabled {
		return nil, fmt.Errorf("registration is disabled")
	}
	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %v", err)
	}

	u, err := s.Store.Client.User.Create().SetUsername(user.Username).SetPassword(string(hashedPassword)).Save(ctx)
	if err != nil {
		if _, ok := err.(*ent.ConstraintError); ok {
			return nil, fmt.Errorf("user already exists")
		}
		return nil, fmt.Errorf("error creating user: %v", err)
	}
	return u, nil
}

func (s *Service) Login(ctx context.Context, uDto user.User) (*ent.User, error) {
	u, err := s.Store.Client.User.Query().Where(entUser.Username(uDto.Username)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(uDto.Password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// uDto = user.User{
	// 	ID:       u.ID,
	// 	Username: u.Username,
	// 	Role:     u.Role,
	// }

	// // generate access token
	// accessToken, exp, err := generateJWTToken(&uDto, time.Now().Add(1*time.Hour), []byte(GetJWTSecret()))
	// if err != nil {
	// 	return nil, fmt.Errorf("error generating access token: %v", err)
	// }

	// // set access token cookie
	// setTokenCookie(c, accessTokenCookieName, accessToken, exp)

	// // generate refresh token
	// refreshToken, exp, err := generateJWTToken(&uDto, time.Now().Add(30*24*time.Hour), []byte(GetJWTRefreshSecret()))
	// if err != nil {
	// 	return nil, fmt.Errorf("error generating refresh token: %v", err)
	// }

	// // set refresh token cookie
	// setTokenCookie(c, refreshTokenCookieName, refreshToken, exp)

	return u, nil
}

func (s *Service) ChangePassword(ctx context.Context, userId uuid.UUID, oldPassword, newPassword string) error {
	// sanity check
	if oldPassword == newPassword {
		return fmt.Errorf("new password must be different from old password")
	}

	// fetch user
	u, err := s.Store.Client.User.Query().Where(entUser.ID(userId)).Only(ctx)
	if err != nil {
		return fmt.Errorf("error getting user: %v", err)
	}

	// validate old password is correct
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(oldPassword))
	if err != nil {
		return fmt.Errorf("invalid credentials")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 14)
	if err != nil {
		return fmt.Errorf("error hashing password: %v", err)
	}

	_, err = u.Update().SetPassword(string(hashedPassword)).Save(ctx)
	if err != nil {
		return fmt.Errorf("error changing password: %v", err)
	}

	return nil
}

// OAuthUserCheck checks if the user from an OIDC flow needs to be created or updated.
func (s *Service) OAuthUserCheck(ctx context.Context, userClaims OIDCCLaims) (*ent.User, error) {
	log.Debug().Msgf("Checking if OAuth user exists: %v", userClaims.PreferredUsername)

	// Check if user exists
	user, err := s.Store.Client.User.Query().Where(entUser.Sub(userClaims.Sub)).Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, fmt.Errorf("failed to query user: %w", err)
		}

		log.Debug().Msgf("OAuth user not found, creating user: %v", userClaims.PreferredUsername)

		// Determine role from groups
		role := utils.UserRole
		for _, group := range userClaims.Groups {
			if strings.HasPrefix(group, "ganymede-") {
				groupRole := strings.TrimPrefix(group, "ganymede-")
				if utils.IsValidRole(groupRole) {
					log.Debug().Msgf("Found Ganymede role in user group %v", group)
					role = utils.Role(groupRole)
					break
				}
			}
		}

		// Create new user
		if _, err := s.Store.Client.User.Create().
			SetSub(userClaims.Sub).
			SetUsername(userClaims.PreferredUsername).
			SetRole(role).
			SetOauth(true).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		return user, nil
	}

	// Determine role from groups
	newRole := utils.UserRole
	for _, group := range userClaims.Groups {
		if strings.HasPrefix(group, "ganymede-") {
			groupRole := strings.TrimPrefix(group, "ganymede-")
			if utils.IsValidRole(groupRole) {
				log.Debug().Msgf("Found Ganymede role in user group %v", group)
				newRole = utils.Role(groupRole)
				break
			}
		}
	}

	// Update existing user
	if _, err := s.Store.Client.User.UpdateOne(user).
		SetUsername(userClaims.PreferredUsername).
		SetRole(newRole).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}
