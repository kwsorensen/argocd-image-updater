package registry

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	"github.com/argoproj-labs/argocd-image-updater/pkg/registry/mocks"
	"github.com/argoproj-labs/argocd-image-updater/pkg/tag"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_GetTags(t *testing.T) {

	t.Run("Check for correctly returned tags with semver sort", func(t *testing.T) {
		regClient := mocks.RegistryClient{}
		regClient.On("Tags", mock.Anything).Return([]string{"1.2.0", "1.2.1", "1.2.2"}, nil)

		ep, err := GetRegistryEndpoint("")
		require.NoError(t, err)

		img := image.NewFromIdentifier("foo/bar:1.2.0")

		tl, err := ep.GetTags(img, &regClient, &image.VersionConstraint{SortMode: image.VersionSortSemVer})
		require.NoError(t, err)
		assert.NotEmpty(t, tl)

		tag, err := ep.Cache.GetTag("foo/bar", "1.2.1")
		require.NoError(t, err)
		assert.Nil(t, tag)
	})

	t.Run("Check for correctly returned tags with filter function applied", func(t *testing.T) {
		regClient := mocks.RegistryClient{}
		regClient.On("Tags", mock.Anything).Return([]string{"1.2.0", "1.2.1", "1.2.2"}, nil)

		ep, err := GetRegistryEndpoint("")
		require.NoError(t, err)

		img := image.NewFromIdentifier("foo/bar:1.2.0")

		tl, err := ep.GetTags(img, &regClient, &image.VersionConstraint{SortMode: image.VersionSortSemVer, MatchFunc: image.MatchFuncNone})
		require.NoError(t, err)
		assert.Empty(t, tl.Tags())

		tag, err := ep.Cache.GetTag("foo/bar", "1.2.1")
		require.NoError(t, err)
		assert.Nil(t, tag)
	})

	t.Run("Check for correctly returned tags with name sort", func(t *testing.T) {

		regClient := mocks.RegistryClient{}
		regClient.On("Tags", mock.Anything).Return([]string{"1.2.0", "1.2.1", "1.2.2"}, nil)

		ep, err := GetRegistryEndpoint("")
		require.NoError(t, err)

		img := image.NewFromIdentifier("foo/bar:1.2.0")

		tl, err := ep.GetTags(img, &regClient, &image.VersionConstraint{SortMode: image.VersionSortName})
		require.NoError(t, err)
		assert.NotEmpty(t, tl)

		tag, err := ep.Cache.GetTag("foo/bar", "1.2.1")
		require.NoError(t, err)
		assert.Nil(t, tag)
	})

	t.Run("Check for correctly returned tags with latest sort", func(t *testing.T) {
		ts := "2006-01-02T15:04:05.999999999Z"
		meta1 := &schema1.SignedManifest{
			Manifest: schema1.Manifest{
				History: []schema1.History{
					{
						V1Compatibility: `{"created":"` + ts + `"}`,
					},
				},
			},
		}
		meta2 := &schema2.DeserializedManifest{
			Manifest: schema2.Manifest{},
		}

		regClient := mocks.RegistryClient{}
		regClient.On("Tags", mock.Anything).Return([]string{"1.2.0", "1.2.1", "1.2.2"}, nil)
		regClient.On("ManifestV1", mock.Anything, mock.Anything).Return(meta1, nil)
		regClient.On("ManifestV2", mock.Anything, mock.Anything).Return(meta2, nil)
		regClient.On("TagMetadata", mock.Anything, mock.Anything).Return(&tag.TagInfo{}, nil)

		ep, err := GetRegistryEndpoint("")
		require.NoError(t, err)
		ep.Cache.ClearCache()

		img := image.NewFromIdentifier("foo/bar:1.2.0")
		tl, err := ep.GetTags(img, &regClient, &image.VersionConstraint{SortMode: image.VersionSortLatest})
		require.NoError(t, err)
		assert.NotEmpty(t, tl)

		tag, err := ep.Cache.GetTag("foo/bar", "1.2.1")
		require.NoError(t, err)
		require.NotNil(t, tag)
		require.Equal(t, "1.2.1", tag.TagName)
	})

	t.Run("Check for correct error handling when manifest contains no history", func(t *testing.T) {
		meta1 := &schema1.SignedManifest{
			Manifest: schema1.Manifest{
				History: []schema1.History{},
			},
		}
		meta2 := &schema2.DeserializedManifest{
			Manifest: schema2.Manifest{},
		}

		regClient := mocks.RegistryClient{}
		regClient.On("Tags", mock.Anything).Return([]string{"1.2.0", "1.2.1", "1.2.2"}, nil)
		regClient.On("ManifestV1", mock.Anything, mock.Anything).Return(meta1, nil)
		regClient.On("ManifestV2", mock.Anything, mock.Anything).Return(meta2, fmt.Errorf("not implemented"))
		regClient.On("TagMetadata", mock.Anything, mock.Anything).Return(nil, nil)

		ep, err := GetRegistryEndpoint("")
		require.NoError(t, err)
		ep.Cache.ClearCache()

		img := image.NewFromIdentifier("foo/bar:1.2.0")
		tl, err := ep.GetTags(img, &regClient, &image.VersionConstraint{SortMode: image.VersionSortLatest})
		require.NoError(t, err)
		assert.Empty(t, tl.Tags())

		tag, err := ep.Cache.GetTag("foo/bar", "1.2.1")
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("Check for correct error handling when manifest contains invalid history", func(t *testing.T) {
		meta1 := &schema1.SignedManifest{
			Manifest: schema1.Manifest{
				History: []schema1.History{
					{
						V1Compatibility: `{"created": {"something": "notastring"}}`,
					},
				},
			},
		}
		meta2 := &schema2.DeserializedManifest{
			Manifest: schema2.Manifest{},
		}

		regClient := mocks.RegistryClient{}
		regClient.On("Tags", mock.Anything).Return([]string{"1.2.0", "1.2.1", "1.2.2"}, nil)
		regClient.On("ManifestV1", mock.Anything, mock.Anything).Return(meta1, nil)
		regClient.On("ManifestV2", mock.Anything, mock.Anything).Return(meta2, fmt.Errorf("not implemented"))
		regClient.On("TagMetadata", mock.Anything, mock.Anything).Return(nil, nil)

		ep, err := GetRegistryEndpoint("")
		require.NoError(t, err)
		ep.Cache.ClearCache()

		img := image.NewFromIdentifier("foo/bar:1.2.0")
		tl, err := ep.GetTags(img, &regClient, &image.VersionConstraint{SortMode: image.VersionSortLatest})
		require.NoError(t, err)
		assert.Empty(t, tl.Tags())

		tag, err := ep.Cache.GetTag("foo/bar", "1.2.1")
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("Check for correct error handling when manifest contains invalid history", func(t *testing.T) {
		meta1 := &schema1.SignedManifest{
			Manifest: schema1.Manifest{
				History: []schema1.History{
					{
						V1Compatibility: `{"something": "something"}`,
					},
				},
			},
		}
		meta2 := &schema2.DeserializedManifest{
			Manifest: schema2.Manifest{},
		}

		regClient := mocks.RegistryClient{}
		regClient.On("Tags", mock.Anything).Return([]string{"1.2.0", "1.2.1", "1.2.2"}, nil)
		regClient.On("ManifestV1", mock.Anything, mock.Anything).Return(meta1, nil)
		regClient.On("ManifestV2", mock.Anything, mock.Anything).Return(meta2, fmt.Errorf("not implemented"))
		regClient.On("TagMetadata", mock.Anything, mock.Anything).Return(nil, nil)

		ep, err := GetRegistryEndpoint("")
		require.NoError(t, err)
		ep.Cache.ClearCache()

		img := image.NewFromIdentifier("foo/bar:1.2.0")
		tl, err := ep.GetTags(img, &regClient, &image.VersionConstraint{SortMode: image.VersionSortLatest})
		require.NoError(t, err)
		assert.Empty(t, tl.Tags())

		tag, err := ep.Cache.GetTag("foo/bar", "1.2.1")
		require.NoError(t, err)
		require.Nil(t, tag)
	})

	t.Run("Check for correct error handling when time stamp cannot be parsed", func(t *testing.T) {
		ts := "invalid"
		meta1 := &schema1.SignedManifest{
			Manifest: schema1.Manifest{
				History: []schema1.History{
					{
						V1Compatibility: `{"created":"` + ts + `"}`,
					},
				},
			},
		}
		meta2 := &schema2.DeserializedManifest{
			Manifest: schema2.Manifest{},
		}

		regClient := mocks.RegistryClient{}
		regClient.On("Tags", mock.Anything).Return([]string{"1.2.0", "1.2.1", "1.2.2"}, nil)
		regClient.On("ManifestV1", mock.Anything, mock.Anything).Return(meta1, nil)
		regClient.On("ManifestV2", mock.Anything, mock.Anything).Return(meta2, fmt.Errorf("not implemented"))
		regClient.On("TagMetadata", mock.Anything, mock.Anything).Return(nil, nil)

		ep, err := GetRegistryEndpoint("")
		require.NoError(t, err)
		ep.Cache.ClearCache()

		img := image.NewFromIdentifier("foo/bar:1.2.0")
		tl, err := ep.GetTags(img, &regClient, &image.VersionConstraint{SortMode: image.VersionSortLatest})
		require.NoError(t, err)
		assert.Empty(t, tl.Tags())

		tag, err := ep.Cache.GetTag("foo/bar", "1.2.1")
		require.NoError(t, err)
		require.Nil(t, tag)
	})

}

func Test_ExpireCredentials(t *testing.T) {
	epYAML := `
registries:
- name: GitHub Container Registry
  api_url: https://ghcr.io
  ping: no
  prefix: ghcr.io
  credentials: env:TEST_CREDS
  credsexpire: 3s
`
	t.Run("Expire credentials", func(t *testing.T) {
		epl, err := ParseRegistryConfiguration(epYAML)
		require.NoError(t, err)
		require.Len(t, epl.Items, 1)

		// New registry configuration
		err = AddRegistryEndpointFromConfig(epl.Items[0])
		require.NoError(t, err)
		ep, err := GetRegistryEndpoint("ghcr.io")
		require.NoError(t, err)
		require.NotEqual(t, 0, ep.CredsExpire)

		// Initial creds
		os.Setenv("TEST_CREDS", "foo:bar")
		err = ep.SetEndpointCredentials(nil)
		assert.NoError(t, err)
		assert.Equal(t, "foo", ep.Username)
		assert.Equal(t, "bar", ep.Password)
		assert.False(t, ep.CredsUpdated.IsZero())

		// Creds should still be cached
		os.Setenv("TEST_CREDS", "bar:foo")
		err = ep.SetEndpointCredentials(nil)
		assert.NoError(t, err)
		assert.Equal(t, "foo", ep.Username)
		assert.Equal(t, "bar", ep.Password)

		// Pretend 5 minutes have passed - creds have expired and are re-read from env
		ep.CredsUpdated = ep.CredsUpdated.Add(time.Minute * -5)
		err = ep.SetEndpointCredentials(nil)
		assert.NoError(t, err)
		assert.Equal(t, "bar", ep.Username)
		assert.Equal(t, "foo", ep.Password)
	})
}
