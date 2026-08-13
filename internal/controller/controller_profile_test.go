package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
)

var _ app.ProfileManagementUseCase = (*stubProfileUseCase)(nil)

type stubProfileUseCase struct {
	seedDefault func(ctx context.Context) ([]dto.IntegrationProfileDTO, error)
}

func (s *stubProfileUseCase) CreateProfile(context.Context, dto.CreateProfileInput) (*dto.IntegrationProfileDTO, error) {
	panic("unexpected CreateProfile")
}

func (s *stubProfileUseCase) UpdateProfile(context.Context, dto.UpdateProfileInput) (*dto.IntegrationProfileDTO, error) {
	panic("unexpected UpdateProfile")
}

func (s *stubProfileUseCase) DeleteProfile(context.Context, uint) error {
	panic("unexpected DeleteProfile")
}

func (s *stubProfileUseCase) GetProfile(context.Context, uint) (*dto.IntegrationProfileDTO, error) {
	panic("unexpected GetProfile")
}

func (s *stubProfileUseCase) ListProfiles(context.Context) ([]dto.IntegrationProfileDTO, error) {
	panic("unexpected ListProfiles")
}

func (s *stubProfileUseCase) SeedDefaultProfiles(ctx context.Context) ([]dto.IntegrationProfileDTO, error) {
	if s.seedDefault != nil {
		return s.seedDefault(ctx)
	}
	return nil, nil
}

func TestSeedDefaultProfiles(t *testing.T) {
	t.Parallel()

	membershipDefault := dto.IntegrationProfileDTO{
		ID:            1,
		ProfileKey:    "membership_default",
		SourceChannel: "default",
		SourceSurface: "membership",
		ConnectorKey:  "eli.local_export",
	}
	defaultErr := errors.New("default seed failed")
	catalogErr := errors.New("catalog demo failed")
	bilibiliErr := errors.New("bilibili demo failed")

	tests := []struct {
		name         string
		seedDefault  func(ctx context.Context) ([]dto.IntegrationProfileDTO, error)
		catalogErr   error
		bilibiliErr  error
		wantErr      error
		wantKey      string
		wantCatalog  bool
		wantBilibili bool
	}{
		{
			name: "membership_default success invokes catalog and bilibili demo seeds",
			seedDefault: func(ctx context.Context) ([]dto.IntegrationProfileDTO, error) {
				return []dto.IntegrationProfileDTO{membershipDefault}, nil
			},
			wantKey:      "membership_default",
			wantCatalog:  true,
			wantBilibili: true,
		},
		{
			name: "default seed failure does not invoke demo seeds",
			seedDefault: func(ctx context.Context) ([]dto.IntegrationProfileDTO, error) {
				return []dto.IntegrationProfileDTO{membershipDefault}, defaultErr
			},
			wantErr: defaultErr,
		},
		{
			name: "catalog demo failure is returned and skips bilibili",
			seedDefault: func(ctx context.Context) ([]dto.IntegrationProfileDTO, error) {
				return []dto.IntegrationProfileDTO{membershipDefault}, nil
			},
			catalogErr:  catalogErr,
			wantErr:     catalogErr,
			wantCatalog: true,
		},
		{
			name: "bilibili demo failure is returned after catalog runs",
			seedDefault: func(ctx context.Context) ([]dto.IntegrationProfileDTO, error) {
				return []dto.IntegrationProfileDTO{membershipDefault}, nil
			},
			bilibiliErr:  bilibiliErr,
			wantErr:      bilibiliErr,
			wantCatalog:  true,
			wantBilibili: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var catalogCalls, bilibiliCalls int
			c := &ProfileController{
				uc: &stubProfileUseCase{seedDefault: tc.seedDefault},
				seedCatalogDemo: func(ctx context.Context) error {
					catalogCalls++
					return tc.catalogErr
				},
				seedBilibiliDemo: func(ctx context.Context) error {
					bilibiliCalls++
					return tc.bilibiliErr
				},
			}

			got, err := c.SeedDefaultProfiles()
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %v, want %v", err, tc.wantErr)
				}
				if got != nil {
					t.Fatalf("failed seed must not return profiles, got %#v", got)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(got) != 1 {
					t.Fatalf("got %d profiles, want 1", len(got))
				}
				if got[0].ProfileKey != tc.wantKey {
					t.Fatalf("ProfileKey = %q, want %q", got[0].ProfileKey, tc.wantKey)
				}
				if got[0].ConnectorKey != "eli.local_export" {
					t.Fatalf("membership_default ConnectorKey = %q, want eli.local_export", got[0].ConnectorKey)
				}
			}

			if (catalogCalls > 0) != tc.wantCatalog {
				t.Fatalf("catalog demo invoked = %d, want invoked=%v", catalogCalls, tc.wantCatalog)
			}
			if (bilibiliCalls > 0) != tc.wantBilibili {
				t.Fatalf("bilibili demo invoked = %d, want invoked=%v", bilibiliCalls, tc.wantBilibili)
			}
		})
	}
}
