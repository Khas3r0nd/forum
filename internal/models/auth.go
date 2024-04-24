package models

import (
	"forum/internal/validator"
	"time"

	"github.com/gofrs/uuid"
)

const (
	GoogleClientID     = "731463570045-2eh8s7p0upas46e1qp310hncbncilnuk.apps.googleusercontent.com"
	GoogleClientSecret = "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"
	GithubClientID     = "6cbc5aceccfb917dc3cd"
	GithubClientSecret = "286b679f651008b8c323d0cc5afb3defa7a4ff54"
)

const (
	GoogleAuthURL = "https://accounts.google.com/o/oauth2/auth"

	GoogleRedirectUrl = "http://localhost:8081/googlecallback"

	GitHubAuthURL = "https://github.com/login/oauth/authorize"

	GithubRedirectUrl = "http://localhost:8081/githubcallback"
)

type GoogleLoginUserData struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Password  string
	Role      string
	Photo     string
	Verified  bool
	Provider  string
	CreatedAt time.Time
	UpdatedAt time.Time
	validator.Validator
}

type GithubLoginUserData struct {
	Id        int
	UserName  string `json:"login"`
	Password  string
	AvatarUrl string `json:"avatar_url"`
	Name      string `json:"name"`
	Role      string
	Provider  string
	validator.Validator
}
