package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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
}

func NewService(store *database.Database) *Service {
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
			Scopes:       []string{oidc.ScopeOpenID, "profile", oidc.ScopeOfflineAccess},
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

func (s *Service) Refresh(c echo.Context, refreshToken string) error {

	tkn, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(GetJWTRefreshSecret()), nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return fmt.Errorf("invalid refresh token")
		}
		return fmt.Errorf("error parsing refresh token: %v", err)
	}

	if claims, ok := tkn.Claims.(jwt.MapClaims); ok && tkn.Valid {
		uID := claims["user_id"].(string)
		uUUID, err := uuid.Parse(uID)
		if err != nil {
			return fmt.Errorf("error parsing user id: %v", err)
		}
		u, err := s.Store.Client.User.Query().Where(entUser.ID(uUUID)).Only(c.Request().Context())
		if err != nil {
			return fmt.Errorf("error getting user: %v", err)
		}

		// generate access token
		accessToken, exp, err := generateJWTToken(&user.User{ID: u.ID, Username: u.Username, Role: u.Role}, time.Now().Add(1*time.Hour), []byte(GetJWTSecret()))
		if err != nil {
			return fmt.Errorf("error generating access token: %v", err)
		}

		// set access token cookie
		setTokenCookie(c, accessTokenCookieName, accessToken, exp)

		return nil
	}

	return err
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

func (s *Service) OAuthUserCheck(c echo.Context, idTokenClaims UserInfo) error {
	// Check if user exists, if not create it or update it
	log.Debug().Msgf("Checking if oauth user exists: %v", idTokenClaims.NickName)
	u, err := s.Store.Client.User.Query().Where(entUser.Sub(idTokenClaims.Sub)).Only(c.Request().Context())
	if err != nil {
		if ent.IsNotFound(err) {
			log.Debug().Msgf("OAuth user not found, creating user: %v", idTokenClaims.NickName)
			role := utils.Role("user")
			// Check what groups the user is in
			var groups []string
			groups = append(groups, idTokenClaims.Groups...)
			// If groups contain ganymede-*
			if len(groups) > 0 {
				for _, group := range groups {
					if strings.Contains(group, "ganymede-") {
						// Extract the role from the group
						groupRole := strings.Replace(group, "ganymede-", "", 1)

						// Check if role is exists in utils.Role enum
						// TODO: make this use utils.Role enum
						if groupRole == "admin" || groupRole == "editor" || groupRole == "archiver" || groupRole == "user" {
							log.Debug().Msgf("Found Ganymede role in user group %v", group)
							role = utils.Role(groupRole)
						}
					}
				}
			}

			// Create user
			_, err = s.Store.Client.User.Create().SetSub(idTokenClaims.Sub).SetUsername(idTokenClaims.NickName).SetRole(utils.Role(role)).SetOauth(true).Save(c.Request().Context())
			if err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			return fmt.Errorf("failed to query user: %w", err)
		}
	} else {
		// Update user
		_, err = s.Store.Client.User.UpdateOne(u).SetUsername(idTokenClaims.NickName).Save(c.Request().Context())
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
	}
	return nil
}
