package identitytags_test

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/identitytags"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

// ── NormalizeIdentityTag ──────────────────────────────────────────────────────

func TestNormalizeIdentityTag(t *testing.T) {
	type args struct {
		platform  string
		tagName   string
		matchMode string
	}
	type want struct {
		platform  string
		tagName   string
		matchMode string
		wantErr   bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "gift_level normal input",
			args: args{platform: "bilibili", tagName: "LV5", matchMode: "gift_level"},
			want: want{platform: "bilibili", tagName: "LV5", matchMode: "gift_level", wantErr: false},
		},
		{
			name: "gift_level empty platform",
			args: args{platform: "", tagName: "LV5", matchMode: "gift_level"},
			want: want{wantErr: true},
		},
		{
			name: "gift_level empty tagName",
			args: args{platform: "bilibili", tagName: "", matchMode: "gift_level"},
			want: want{wantErr: true},
		},
		{
			name: "platform_all normal",
			args: args{platform: "douyin", tagName: "should_be_cleared", matchMode: "platform_all"},
			want: want{platform: "douyin", tagName: "", matchMode: "platform_all", wantErr: false},
		},
		{
			name: "platform_all empty platform",
			args: args{platform: "", tagName: "", matchMode: "platform_all"},
			want: want{wantErr: true},
		},
		{
			name: "wave_all normal",
			args: args{platform: "bilibili", tagName: "LV5", matchMode: "wave_all"},
			want: want{platform: "", tagName: "", matchMode: "wave_all", wantErr: false},
		},
		{
			name: "user_member rejected",
			args: args{platform: "bilibili", tagName: "uid123", matchMode: "user_member"},
			want: want{wantErr: true},
		},
		{
			name: "unknown matchMode rejected",
			args: args{platform: "bilibili", tagName: "LV5", matchMode: "foobar"},
			want: want{wantErr: true},
		},
		{
			name: "empty matchMode defaults to gift_level",
			args: args{platform: "bilibili", tagName: "LV5", matchMode: ""},
			want: want{platform: "bilibili", tagName: "LV5", matchMode: "gift_level", wantErr: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPlatform, gotTagName, gotMatchMode, err := identitytags.NormalizeIdentityTag(tt.args.platform, tt.args.tagName, tt.args.matchMode)
			if (err != nil) != tt.want.wantErr {
				t.Fatalf("NormalizeIdentityTag() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if tt.want.wantErr {
				return
			}
			if gotPlatform != tt.want.platform {
				t.Errorf("platform: got %q, want %q", gotPlatform, tt.want.platform)
			}
			if gotTagName != tt.want.tagName {
				t.Errorf("tagName: got %q, want %q", gotTagName, tt.want.tagName)
			}
			if gotMatchMode != tt.want.matchMode {
				t.Errorf("matchMode: got %q, want %q", gotMatchMode, tt.want.matchMode)
			}
		})
	}
}

// ── ProductTagMatchesWaveMember ───────────────────────────────────────────────

func TestProductTagMatchesWaveMember(t *testing.T) {
	type args struct {
		tag model.ProductTag
		wm  model.WaveMember
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// gift_level: matches
		{
			name: "gift_level match",
			args: args{
				tag: model.ProductTag{Platform: "bilibili", TagName: "LV5", MatchMode: "gift_level", TagType: "identity"},
				wm:  model.WaveMember{Platform: "bilibili", GiftLevel: "LV5"},
			},
			want: true,
		},
		// gift_level: different platform
		{
			name: "gift_level different platform",
			args: args{
				tag: model.ProductTag{Platform: "bilibili", TagName: "LV5", MatchMode: "gift_level", TagType: "identity"},
				wm:  model.WaveMember{Platform: "douyin", GiftLevel: "LV5"},
			},
			want: false,
		},
		// gift_level: different tagName
		{
			name: "gift_level different tagName",
			args: args{
				tag: model.ProductTag{Platform: "bilibili", TagName: "LV5", MatchMode: "gift_level", TagType: "identity"},
				wm:  model.WaveMember{Platform: "bilibili", GiftLevel: "LV3"},
			},
			want: false,
		},
		// platform_all: same platform matches
		{
			name: "platform_all same platform",
			args: args{
				tag: model.ProductTag{Platform: "douyin", TagName: "", MatchMode: "platform_all", TagType: "identity"},
				wm:  model.WaveMember{Platform: "douyin", GiftLevel: "LV3"},
			},
			want: true,
		},
		// platform_all: different platform does not match
		{
			name: "platform_all different platform",
			args: args{
				tag: model.ProductTag{Platform: "douyin", TagName: "", MatchMode: "platform_all", TagType: "identity"},
				wm:  model.WaveMember{Platform: "bilibili", GiftLevel: "LV3"},
			},
			want: false,
		},
		// wave_all: always matches
		{
			name: "wave_all matches any member",
			args: args{
				tag: model.ProductTag{Platform: "", TagName: "", MatchMode: "wave_all", TagType: "identity"},
				wm:  model.WaveMember{Platform: "bilibili", GiftLevel: "LV5"},
			},
			want: true,
		},
		// user_member: default branch → false
		{
			name: "user_member matchMode returns false",
			args: args{
				tag: model.ProductTag{Platform: "bilibili", TagName: "uid123", MatchMode: "user_member", TagType: "user"},
				wm:  model.WaveMember{Platform: "bilibili", GiftLevel: "LV5"},
			},
			want: false,
		},
		// default/unknown matchMode → false
		{
			name: "unknown matchMode returns false",
			args: args{
				tag: model.ProductTag{Platform: "bilibili", TagName: "LV5", MatchMode: "foobar", TagType: "identity"},
				wm:  model.WaveMember{Platform: "bilibili", GiftLevel: "LV5"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := identitytags.ProductTagMatchesWaveMember(tt.args.tag, tt.args.wm)
			if got != tt.want {
				t.Errorf("ProductTagMatchesWaveMember() = %v, want %v", got, tt.want)
			}
		})
	}
}
