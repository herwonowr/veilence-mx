package persistent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// SSOConfigRepo implements usecase.SSOConfigRepository using GORM.
type SSOConfigRepo struct {
	db        *gorm.DB
	encryptor SecretEncryptor
}

// SecretEncryptor encrypts and decrypts OAuth client secrets at the repo layer.
type SecretEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// noopEncryptor is a passthrough used when no encryption key is configured.
type noopEncryptor struct{}

func (noopEncryptor) Encrypt(plaintext string) (string, error)  { return plaintext, nil }
func (noopEncryptor) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }

// NewSSOConfigRepo creates a new SSOConfigRepo.
// If encryptor is nil, a noop encryptor is used (no encryption).
func NewSSOConfigRepo(db *gorm.DB, encryptor SecretEncryptor) *SSOConfigRepo {
	if encryptor == nil {
		encryptor = noopEncryptor{}
	}
	return &SSOConfigRepo{db: db, encryptor: encryptor}
}

func (r *SSOConfigRepo) FindAll(ctx context.Context) ([]entity.SSOConfig, error) {
	var ms []SSOConfig
	if err := r.db.WithContext(ctx).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("SSOConfigRepo.FindAll: %w", err)
	}
	result := make([]entity.SSOConfig, len(ms))
	for i := range ms {
		d, err := r.ssoConfigToDomain(&ms[i])
		if err != nil {
			return nil, fmt.Errorf("SSOConfigRepo.FindAll: decrypt: %w", err)
		}
		result[i] = *d
	}
	return result, nil
}

func (r *SSOConfigRepo) FindByID(ctx context.Context, id string) (*entity.SSOConfig, error) {
	var m SSOConfig
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("sso config %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("SSOConfigRepo.FindByID: %w", err)
	}
	return r.ssoConfigToDomain(&m)
}

func (r *SSOConfigRepo) FindEnabled(ctx context.Context) ([]entity.SSOConfig, error) {
	var ms []SSOConfig
	if err := r.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("SSOConfigRepo.FindEnabled: %w", err)
	}
	result := make([]entity.SSOConfig, len(ms))
	for i := range ms {
		d, err := r.ssoConfigToDomain(&ms[i])
		if err != nil {
			return nil, fmt.Errorf("SSOConfigRepo.FindEnabled: decrypt: %w", err)
		}
		result[i] = *d
	}
	return result, nil
}

func (r *SSOConfigRepo) FindBySAMLEntityID(ctx context.Context, entityID string) (*entity.SSOConfig, error) {
	var m SSOConfig
	if err := r.db.WithContext(ctx).Where("saml_entity_id = ? AND provider = ?", entityID, "saml").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("sso config by entity ID %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("SSOConfigRepo.FindBySAMLEntityID: %w", err)
	}
	return r.ssoConfigToDomain(&m)
}

func (r *SSOConfigRepo) FindByProvider(ctx context.Context, provider entity.SSOProvider) (*entity.SSOConfig, error) {
	var m SSOConfig
	if err := r.db.WithContext(ctx).Where("provider = ?", string(provider)).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("sso config by provider %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("SSOConfigRepo.FindByProvider: %w", err)
	}
	return r.ssoConfigToDomain(&m)
}

func (r *SSOConfigRepo) Create(ctx context.Context, config *entity.SSOConfig) error {
	m, err := r.ssoConfigToModel(config)
	if err != nil {
		return fmt.Errorf("SSOConfigRepo.Create: encrypt: %w", err)
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("SSOConfigRepo.Create: %w", err)
	}
	config.ID = m.ID
	config.CreatedAt = m.CreatedAt
	config.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *SSOConfigRepo) Update(ctx context.Context, config *entity.SSOConfig) error {
	m, err := r.ssoConfigToModel(config)
	if err != nil {
		return fmt.Errorf("SSOConfigRepo.Update: encrypt: %w", err)
	}
	m.UpdatedAt = time.Now()
	result := r.db.WithContext(ctx).Where("id = ?", m.ID).Save(m)
	if result.Error != nil {
		return fmt.Errorf("SSOConfigRepo.Update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("SSOConfigRepo.Update: %w", entity.ErrNotFound)
	}
	config.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *SSOConfigRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&SSOConfig{})
	if result.Error != nil {
		return fmt.Errorf("SSOConfigRepo.Delete: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("SSOConfigRepo.Delete: %w", entity.ErrNotFound)
	}
	return nil
}

// --- Converters ---

func (r *SSOConfigRepo) ssoConfigToDomain(m *SSOConfig) (*entity.SSOConfig, error) {
	secret := ""
	if m.OAuthClientSecretEnc != "" {
		var err error
		secret, err = r.encryptor.Decrypt(m.OAuthClientSecretEnc)
		if err != nil {
			return nil, err
		}
	}
	return &entity.SSOConfig{
		ID:                 m.ID,
		Provider:           entity.SSOProvider(m.Provider),
		DisplayName:        m.DisplayName,
		IsEnabled:          m.IsEnabled,
		AllowedDomains:     splitDomains(m.AllowedDomains),
		AutoCreateUser:     m.AutoCreateUser,
		SAMLEntityID:       m.SAMLEntityID,
		SAMLSsoURL:         m.SAMLSsoURL,
		SAMLCertificate:    m.SAMLCertificate,
		SAMLAttrEmail:      m.SAMLAttrEmail,
		SAMLAttrFirstName:  m.SAMLAttrFirstName,
		SAMLAttrLastName:   m.SAMLAttrLastName,
		OAuthClientID:      m.OAuthClientID,
		OAuthClientSecret:  secret,
		GoogleHostedDomain: m.GoogleHostedDomain,
		GitHubOrgs:         splitDomains(m.GitHubOrgs),
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}, nil
}

func (r *SSOConfigRepo) ssoConfigToModel(d *entity.SSOConfig) (*SSOConfig, error) {
	encSecret := ""
	if d.OAuthClientSecret != "" {
		var err error
		encSecret, err = r.encryptor.Encrypt(d.OAuthClientSecret)
		if err != nil {
			return nil, err
		}
	}
	return &SSOConfig{
		ID:                   d.ID,
		Provider:             string(d.Provider),
		DisplayName:          d.DisplayName,
		IsEnabled:            d.IsEnabled,
		AllowedDomains:       joinDomains(d.AllowedDomains),
		AutoCreateUser:       d.AutoCreateUser,
		SAMLEntityID:         d.SAMLEntityID,
		SAMLSsoURL:           d.SAMLSsoURL,
		SAMLCertificate:      d.SAMLCertificate,
		SAMLAttrEmail:        d.SAMLAttrEmail,
		SAMLAttrFirstName:    d.SAMLAttrFirstName,
		SAMLAttrLastName:     d.SAMLAttrLastName,
		OAuthClientID:        d.OAuthClientID,
		OAuthClientSecretEnc: encSecret,
		GoogleHostedDomain:   d.GoogleHostedDomain,
		GitHubOrgs:           joinDomains(d.GitHubOrgs),
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
	}, nil
}

// --- Helpers ---

func splitDomains(s string) []string {
	if s == "" || s == "[]" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return []string{}
	}
	if result == nil {
		return []string{}
	}
	return result
}

func joinDomains(domains []string) string {
	if domains == nil {
		domains = []string{}
	}
	b, _ := json.Marshal(domains)
	return string(b)
}
