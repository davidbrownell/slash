package v2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/boojack/slash/api/auth"
	apiv2pb "github.com/boojack/slash/proto/gen/api/v2"
	storepb "github.com/boojack/slash/proto/gen/store"
	"github.com/boojack/slash/server/metric"
	"github.com/boojack/slash/server/service/license"
	"github.com/boojack/slash/store"
)

func (s *APIV2Service) SignIn(ctx context.Context, request *apiv2pb.SignInRequest) (*apiv2pb.SignInResponse, error) {
	user, err := s.Store.GetUser(ctx, &store.FindUser{
		Email: &request.Email,
	})
	if err != nil {
		return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to find user by email %s", request.Email))
	}
	if user == nil {
		return nil, status.Errorf(http.StatusUnauthorized, fmt.Sprintf("user not found with email %s", request.Email))
	} else if user.RowStatus == store.Archived {
		return nil, status.Errorf(http.StatusForbidden, fmt.Sprintf("user has been archived with email %s", request.Email))
	}

	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		return nil, status.Errorf(http.StatusUnauthorized, "unmatched email and password")
	}

	accessToken, err := auth.GenerateAccessToken(user.Email, user.ID, time.Now().Add(auth.AccessTokenDuration), []byte(s.Secret))
	if err != nil {
		return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to generate tokens, err: %s", err))
	}
	if err := s.UpsertAccessTokenToStore(ctx, user, accessToken, "user login"); err != nil {
		return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to upsert access token to store, err: %s", err))
	}

	metric.Enqueue("user sign in")
	return &apiv2pb.SignInResponse{
		User:        convertUserFromStore(user),
		AccessToken: accessToken,
	}, nil
}

func (s *APIV2Service) SignUp(ctx context.Context, request *apiv2pb.SignUpRequest) (*apiv2pb.SignUpResponse, error) {
	enableSignUpSetting, err := s.Store.GetWorkspaceSetting(ctx, &store.FindWorkspaceSetting{
		Key: storepb.WorkspaceSettingKey_WORKSAPCE_SETTING_ENABLE_SIGNUP,
	})
	if err != nil {
		return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to get workspace setting, err: %s", err))
	}
	if enableSignUpSetting != nil && !enableSignUpSetting.GetEnableSignup() {
		return nil, status.Errorf(http.StatusForbidden, "sign up is not allowed")
	}

	if !s.LicenseService.IsFeatureEnabled(license.FeatureTypeUnlimitedAccounts) {
		userList, err := s.Store.ListUsers(ctx, &store.FindUser{})
		if err != nil {
			return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to list users, err: %s", err))
		}
		if len(userList) >= 5 {
			return nil, status.Errorf(http.StatusBadRequest, "maximum number of users reached")
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to generate password hash, err: %s", err))
	}

	create := &store.User{
		Email:        request.Email,
		Nickname:     request.Nickname,
		PasswordHash: string(passwordHash),
	}
	existingUsers, err := s.Store.ListUsers(ctx, &store.FindUser{})
	if err != nil {
		return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to list users, err: %s", err))
	}
	// The first user to sign up is an admin by default.
	if len(existingUsers) == 0 {
		create.Role = store.RoleAdmin
	} else {
		create.Role = store.RoleUser
	}

	user, err := s.Store.CreateUser(ctx, create)
	if err != nil {
		return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to create user, err: %s", err))
	}

	accessToken, err := auth.GenerateAccessToken(user.Email, user.ID, time.Now().Add(auth.AccessTokenDuration), []byte(s.Secret))
	if err != nil {
		return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to generate tokens, err: %s", err))
	}
	if err := s.UpsertAccessTokenToStore(ctx, user, accessToken, "user login"); err != nil {
		return nil, status.Errorf(http.StatusInternalServerError, fmt.Sprintf("failed to upsert access token to store, err: %s", err))
	}

	metric.Enqueue("user sign up")
	return &apiv2pb.SignUpResponse{
		User:        convertUserFromStore(user),
		AccessToken: accessToken,
	}, nil
}

func (*APIV2Service) SignOut(ctx context.Context, _ *apiv2pb.SignOutRequest) (*apiv2pb.SignOutResponse, error) {
	if err := grpc.SetHeader(ctx, metadata.New(map[string]string{
		auth.AccessTokenCookieName: "",
	})); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set grpc header, error: %v", err)
	}
	return &apiv2pb.SignOutResponse{}, nil
}
