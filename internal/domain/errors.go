package domain

import "errors"

// Common domain errors
var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInternalServer    = errors.New("internal server error")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken        = errors.New("email already taken")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
	ErrNoActiveStatus    = errors.New("no active status found")
)

// Connection-specific errors with clear messages
var (
	ErrSelfConnection       = errors.New("cannot send connection request to yourself")
	ErrDuplicatePending     = errors.New("a connection request already exists")
	ErrAlreadyConnected     = errors.New("you are already connected")
	ErrConnectionBlocked    = errors.New("connection request cannot be sent")
)
