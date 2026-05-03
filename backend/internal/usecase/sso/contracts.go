package sso

import "github.com/veilence/veilence-mx/backend/internal/usecase"

// Type aliases for interfaces defined in the shared usecase package.
// This avoids import cycles: repo/persistent can import usecase (not usecase/sso).
type SSOConfigRepository = usecase.SSOConfigRepository
type UserIdentityRepository = usecase.UserIdentityRepository
type SSOStateRepository = usecase.SSOStateRepository
type SAMLProvider = usecase.SAMLProvider
type OAuthTokenExchanger = usecase.OAuthTokenExchanger
type AuditLogger = usecase.AuditLogger
type SAMLMetadataFetcher = usecase.SAMLMetadataFetcher
type AuthSessionCreator = usecase.AuthSessionCreator
type TokenPair = usecase.TokenPair
type SessionRepository = usecase.SessionRepository
type RefreshTokenRepository = usecase.RefreshTokenRepository
