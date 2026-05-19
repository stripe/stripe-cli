package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func computeChallenge(salt string, number int64) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(strconv.FormatInt(number, 10)))
	return hex.EncodeToString(h.Sum(nil))
}

func TestSolveChallenge_KnownSolution(t *testing.T) {
	salt := "testsalt"
	expected := int64(42)
	challenge := computeChallenge(salt, expected)

	result, err := SolveChallenge(context.Background(), "SHA-256", challenge, salt)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestSolveChallenge_Zero(t *testing.T) {
	salt := "zero"
	challenge := computeChallenge(salt, int64(0))

	result, err := SolveChallenge(context.Background(), "SHA-256", challenge, salt)
	require.NoError(t, err)
	assert.Equal(t, int64(0), result)
}

func TestSolveChallenge_UnsupportedAlgorithm(t *testing.T) {
	_, err := SolveChallenge(context.Background(), "MD5", "abc", "salt")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedAlgorithm)
}

func TestSolveChallenge_InvalidChallengeHex(t *testing.T) {
	_, err := SolveChallenge(context.Background(), "SHA-256", "not-hex!", "salt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid challenge hex")
}

func TestSolveChallenge_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := SolveChallenge(ctx, "SHA-256", "0000000000000000000000000000000000000000000000000000000000000000", "salt")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSolveChallenge_ContextCancelledMidSolve(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Use a challenge that will never be solved (requires iterating past timeout)
	unsolvable := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	_, err := SolveChallenge(ctx, "SHA-256", unsolvable, "will-never-match")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSolveChallenge_AlgorithmVariants(t *testing.T) {
	salt := "variant"
	expected := int64(7)
	challenge := computeChallenge(salt, expected)

	tests := []string{"SHA-256", "sha-256", "SHA256", "sha256"}
	for _, algo := range tests {
		t.Run(algo, func(t *testing.T) {
			result, err := SolveChallenge(context.Background(), algo, challenge, salt)
			require.NoError(t, err)
			assert.Equal(t, expected, result)
		})
	}
}

func TestClient_GetChallenge_Success(t *testing.T) {
	expected := &ChallengeResponse{
		Algorithm: "SHA-256",
		Challenge: "abc123",
		Salt:      "mysalt",
		Signature: "sig456",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/keys/challenge", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("User-Agent"))

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "test@example.com", body["email"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.GetChallenge(context.Background(), "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestClient_GetChallenge_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetChallenge(context.Background(), "test@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClient_GetChallenge_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not json")
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetChallenge(context.Background(), "test@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestClient_GetChallenge_MissingFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"algorithm": "SHA-256"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetChallenge(context.Background(), "test@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid challenge response")
}

func TestClient_Provision_Success(t *testing.T) {
	serverResp := map[string]interface{}{
		"secret_key":      "sk_test_abc123",
		"publishable_key": "pk_test_xyz789",
		"claim_url":       "https://dashboard.stripe.com/claim_sandbox/token",
		"expires_at":      "2026-04-25T03:19:09.000Z",
		"account_id":      "acct_123",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/keys/provision", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body ProvisionRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "test@example.com", body.Email)
		assert.Equal(t, int64(42), body.Number)
		assert.Equal(t, "sig", body.Signature)
		assert.Equal(t, "SHA-256", body.Algorithm)
		assert.Equal(t, "challenge", body.Challenge)
		assert.Equal(t, "salt", body.Salt)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(serverResp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Provision(context.Background(), ProvisionRequest{
		Algorithm: "SHA-256",
		Challenge: "challenge",
		Salt:      "salt",
		Signature: "sig",
		Number:    42,
		Email:     "test@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "sk_test_abc123", result.GetSecretKey())
	assert.Equal(t, "pk_test_xyz789", result.GetPublishableKey())
	assert.Equal(t, "https://dashboard.stripe.com/claim_sandbox/token", result.GetClaimURL())
	assert.Equal(t, "acct_123", result.GetAccountID())
}

func TestClient_Provision_WithName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body ProvisionRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Test User", body.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"secret_key": "sk_test_x"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Provision(context.Background(), ProvisionRequest{
		Email: "test@example.com",
		Name:  "Test User",
	})
	require.NoError(t, err)
}

func TestClient_Provision_NameOmittedWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]interface{}
		json.NewDecoder(r.Body).Decode(&raw)
		_, hasName := raw["name"]
		assert.False(t, hasName, "name field should be omitted when empty")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"secret_key": "sk_test_x"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Provision(context.Background(), ProvisionRequest{
		Email: "test@example.com",
		Name:  "",
	})
	require.NoError(t, err)
}

func TestClient_Provision_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "rate limited")
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Provision(context.Background(), ProvisionRequest{Email: "test@example.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

func TestGitConfigFunc_Replaceable(t *testing.T) {
	original := GitConfigFunc
	defer func() { GitConfigFunc = original }()

	GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "mock@example.com"
		}
		return ""
	}

	assert.Equal(t, "mock@example.com", GitConfigFunc("user.email"))
	assert.Equal(t, "", GitConfigFunc("user.name"))
}

func TestClient_FullFlow(t *testing.T) {
	salt := "integration-salt"
	secretNumber := int64(17)
	challenge := computeChallenge(salt, secretNumber)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/keys/challenge":
			json.NewEncoder(w).Encode(ChallengeResponse{
				Algorithm: "SHA-256",
				Challenge: challenge,
				Salt:      salt,
				Signature: "test-sig",
			})
		case "/keys/provision":
			var req ProvisionRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Number != secretNumber {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, "invalid solution")
				return
			}
			assert.Equal(t, "test-sig", req.Signature)
			assert.Equal(t, "user@example.com", req.Email)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"secret_key":      "sk_test_provisioned",
				"publishable_key": "pk_test_provisioned",
				"claim_url":       "https://dashboard.stripe.com/claim_sandbox/abc",
				"expires_at":      "2026-05-10T00:00:00Z",
				"account_id":      "acct_test_123",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	challengeResp, err := client.GetChallenge(context.Background(), "user@example.com")
	require.NoError(t, err)

	solution, err := SolveChallenge(context.Background(), challengeResp.Algorithm, challengeResp.Challenge, challengeResp.Salt)
	require.NoError(t, err)
	assert.Equal(t, secretNumber, solution)

	result, err := client.Provision(context.Background(), ProvisionRequest{
		Algorithm: challengeResp.Algorithm,
		Challenge: challengeResp.Challenge,
		Salt:      challengeResp.Salt,
		Signature: challengeResp.Signature,
		Number:    solution,
		Email:     "user@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "sk_test_provisioned", result.GetSecretKey())
	assert.Equal(t, "pk_test_provisioned", result.GetPublishableKey())
	assert.Equal(t, "acct_test_123", result.GetAccountID())
}
