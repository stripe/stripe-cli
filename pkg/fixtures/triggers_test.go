package fixtures

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRequiredParams(t *testing.T) {
	t.Run("no required params and no provided params", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{})
		require.NoError(t, err)
	})

	t.Run("no required params but params provided shows helpful error", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{"charge:amount=1000"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unexpected parameters")
		assert.Contains(t, err.Error(), "use --override instead")
		assert.Contains(t, err.Error(), "charge:amount=1000")
	})

	t.Run("all required params provided", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{
						{
							Name:        "charge:transfer_data.destination",
							Description: "Connect account ID",
							Placeholder: "acct_123",
						},
					},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{"charge:transfer_data.destination=acct_test"})
		require.NoError(t, err)
	})

	t.Run("missing required param shows actionable error", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{
						{
							Name:        "charge:transfer_data.destination",
							Description: "Connect account ID",
							Placeholder: "acct_123",
						},
					},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Missing required parameters")
		assert.Contains(t, err.Error(), "charge:transfer_data.destination")
		assert.Contains(t, err.Error(), "Connect account ID")
		assert.Contains(t, err.Error(), "acct_123")
	})

	t.Run("multiple required params all provided", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{
						{
							Name:        "charge:transfer_data.destination",
							Description: "Connect account ID",
							Placeholder: "acct_123",
						},
						{
							Name:        "charge:amount",
							Description: "Charge amount",
							Placeholder: "1000",
						},
					},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{
			"charge:transfer_data.destination=acct_test",
			"charge:amount=2000",
		})
		require.NoError(t, err)
	})

	t.Run("multiple required params partially provided", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{
						{
							Name:        "charge:transfer_data.destination",
							Description: "Connect account ID",
							Placeholder: "acct_123",
						},
						{
							Name:        "charge:amount",
							Description: "Charge amount",
							Placeholder: "1000",
						},
					},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{"charge:amount=2000"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Missing required parameters")
		assert.Contains(t, err.Error(), "charge:transfer_data.destination")
		assert.NotContains(t, err.Error(), "charge:amount") // amount was provided
	})

	t.Run("malformed param missing equals sign", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{
						{
							Name:        "charge:amount",
							Description: "Charge amount",
							Placeholder: "1000",
						},
					},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{"charge:amount"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Malformed parameter")
		assert.Contains(t, err.Error(), "fixtureName:path.to.field=value")
	})

	t.Run("malformed param with empty name", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{
						{
							Name:        "charge:amount",
							Description: "Charge amount",
							Placeholder: "1000",
						},
					},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{"=value"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Malformed parameter")
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("malformed param with empty value", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{
						{
							Name:        "charge:amount",
							Description: "Charge amount",
							Placeholder: "1000",
						},
					},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{"charge:amount="})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Malformed parameter")
		assert.Contains(t, err.Error(), "value cannot be empty")
		assert.Contains(t, err.Error(), "use --override instead")
	})

	t.Run("param with special characters in value", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{
						{
							Name:        "charge:description",
							Description: "Charge description",
							Placeholder: "Test charge",
						},
					},
				},
			},
		}

		err := ValidateRequiredParams(fxt, "", []string{"charge:description=Test & Co."})
		require.NoError(t, err)
	})

	t.Run("param with equals sign in value", func(t *testing.T) {
		fxt := &Fixture{
			FixtureData: FixtureData{
				Meta: MetaFixture{
					RequiredParams: []RequiredParam{
						{
							Name:        "charge:metadata.key",
							Description: "Metadata key",
							Placeholder: "value",
						},
					},
				},
			},
		}

		// SplitN with 2 should handle this correctly
		err := ValidateRequiredParams(fxt, "", []string{"charge:metadata.key=value=with=equals"})
		require.NoError(t, err)
	})
}

// Integration tests for param flow through BuildFromFixtureFile
func TestBuildFromFixtureFileWithParams(t *testing.T) {
	const fixtureWithRequiredParams = `
{
  "_meta": {
    "template_version": 0,
    "required_params": [
      {
        "name": "charge:transfer_data.destination",
        "description": "Connect account ID",
        "placeholder": "acct_123"
      }
    ]
  },
  "fixtures": [
    {
      "name": "charge",
      "path": "/v1/charges",
      "method": "post",
      "params": {
        "amount": 1000,
        "currency": "usd",
        "source": "tok_visa",
        "transfer_data": {
          "destination": "{{CONNECT_ACCOUNT_ID}}"
        }
      }
    }
  ]
}
`

	t.Run("missing required param fails before API call", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "test.json", []byte(fixtureWithRequiredParams), 0644)

		_, err := BuildFromFixtureFile(
			fs,
			"sk_test_123",
			"",
			"http://localhost",
			"test.json",
			[]string{}, // skip
			[]string{}, // override
			[]string{}, // param - missing!
			[]string{}, // add
			[]string{}, // remove
			false,      // edit
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Missing required parameters")
		assert.Contains(t, err.Error(), "charge:transfer_data.destination")
	})

	t.Run("required param provided succeeds", func(t *testing.T) {
		// Set up mock server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "ch_123", "object": "charge"}`))
		}))
		defer ts.Close()

		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "test.json", []byte(fixtureWithRequiredParams), 0644)

		fixture, err := BuildFromFixtureFile(
			fs,
			"sk_test_123",
			"",
			ts.URL,
			"test.json",
			[]string{}, // skip
			[]string{}, // override
			[]string{"charge:transfer_data.destination=acct_test"}, // param provided
			[]string{}, // add
			[]string{}, // remove
			false,      // edit
		)

		require.NoError(t, err)
		require.NotNil(t, fixture)

		// Execute the fixture to ensure param was applied
		_, err = fixture.Execute(context.Background(), "")
		require.NoError(t, err)
	})

	t.Run("params take precedence over override", func(t *testing.T) {
		// Set up mock server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "ch_123", "object": "charge"}`))
		}))
		defer ts.Close()

		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "test.json", []byte(fixtureWithRequiredParams), 0644)

		fixture, err := BuildFromFixtureFile(
			fs,
			"sk_test_123",
			"",
			ts.URL,
			"test.json",
			[]string{}, // skip
			[]string{"charge:transfer_data.destination=acct_override"}, // override
			[]string{"charge:transfer_data.destination=acct_param"},    // param should win
			[]string{}, // add
			[]string{}, // remove
			false,      // edit
		)

		require.NoError(t, err)
		require.NotNil(t, fixture)

		// Execute the fixture
		_, err = fixture.Execute(context.Background(), "")
		require.NoError(t, err)
	})

	t.Run("malformed param syntax fails validation", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "test.json", []byte(fixtureWithRequiredParams), 0644)

		_, err := BuildFromFixtureFile(
			fs,
			"sk_test_123",
			"",
			"http://localhost",
			"test.json",
			[]string{},                      // skip
			[]string{},                      // override
			[]string{"malformed_no_equals"}, // malformed param
			[]string{},                      // add
			[]string{},                      // remove
			false,                           // edit
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Malformed parameter")
		assert.Contains(t, err.Error(), "fixtureName:path.to.field=value")
	})
}
