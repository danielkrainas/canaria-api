package token

import (
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/docker/libtrust"

	"github.com/danielkrainas/canaria-api/auth"
	"github.com/danielkrainas/canaria-api/context"
)

type accessSet map[auth.Resource]actionSet

var (
	ErrInsufficientScope = errors.New("insufficient scope")
	ErrTokenRequired     = errors.New("authorization token required")
	ErrMalformedToken    = errors.New("malformed token")
	ErrInvalidToken      = errors.New("invalid token")
)

type authChallenge struct {
	err       error
	realm     string
	service   string
	accessSet accessSet
}

var _ auth.Challenge = authChallenge{}

type authStrategy struct {
	realm       string
	issuer      string
	service     string
	rootCerts   *x509.CertPool
	trustedKeys map[string]libtrust.PublicKey
}

type tokenAccessOptions struct {
	realm          string
	issuer         string
	service        string
	rootCertBundle string
}

type VerifyOptions struct {
	TrustedIssuers    []string
	AcceptedAudiences []string
	Roots             *x509.CertPool
	TrustedKeys       map[string]libtrust.PublicKey
}

func newAccessSet(accessItems ...auth.Access) accessSet {
	accessSet := make(accessSet, len(accessItems))

	for _, access := range accessItems {
		res := auth.Resource{
			Type: access.Type,
			Name: access.Name,
		}

		set, exists := accessSet[res]
		if !exists {
			set = newActionSet()
			accessSet[res] = set
		}

		set.add(access.Action)
	}

	return accessSet
}

func (s accessSet) contains(access auth.Access) bool {
	if actionSet, ok := s[access.Resource]; ok {
		return actionSet.contains(access.Action)
	}

	return false
}

func (s accessSet) scopeParam() string {
	scopes := make([]string, 0, len(s))
	for res, actionSet := range s {
		actions := strings.Join(actionSet.keys(), ",")
		scopes = append(scopes, fmt.Sprintf("%s:%s:%s", res.Type, res.Name, actions))
	}

	return strings.Join(scopes, " ")
}

func (ac authChallenge) Error() string {
	return ac.err.Error()
}

func (ac authChallenge) Status() int {
	return http.StatusUnauthorized
}

func (ac authChallenge) challengeParams() string {
	str := fmt.Sprintf("Bearer realm=%q,service=%q", ac.realm, ac.service)
	if scope := ac.accessSet.scopeParam(); scope != "" {
		str = fmt.Sprintf("%s,scope=%q", str, scope)
	}

	if ac.err == ErrInvalidToken || ac.err == ErrMalformedToken {
		str = fmt.Sprintf("%s,error=%q", str, "invalid_token")
	} else if ac.err == ErrInsufficientScope {
		str = fmt.Sprintf("%s,error=%q", str, "insufficient_scope")
	}

	return str
}

func (ac authChallenge) SetHeaders(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", ac.challengeParams())
}

func checkOptions(options map[string]interface{}) (tokenAccessOptions, error) {
	var opts tokenAccessOptions
	keys := []string{"realm", "issuer", "service", "rootcertbundle"}
	vals := make([]string, 0, len(keys))
	for _, key := range keys {
		val, ok := options[key].(string)
		if !ok {
			return opts, fmt.Errorf("token auth requires a valid option string: %q", key)
		}

		vals = append(vals, val)
	}

	opts.realm = vals[0]
	opts.issuer = vals[1]
	opts.service = vals[2]
	opts.rootCertBundle = vals[3]
	return opts, nil
}

func newAuthStrategy(options map[string]interface{}) (auth.AuthStrategy, error) {
	config, err := checkOptions(options)
	if err != nil {
		return nil, err
	}

	fp, err := os.Open(config.rootCertBundle)
	if err != nil {
		return nil, fmt.Errorf("unable to open auth root certificate bundle file %q: %s", config.rootCertBundle, err)
	}

	defer fp.Close()
	rawCertBundle, err := ioutil.ReadAll(fp)
	if err != nil {
		return nil, fmt.Errorf("unable to read token auth root certificate bundle file %q: %s", config.rootCertBundle, err)
	}

	var rootCerts []*x509.Certificate
	pemBlock, rawCertBundle := pem.Decode(rawCertBundle)
	for pemBlock != nil {
		cert, err := x509.ParseCertificate(pemBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("unable to parse token auth root certificate: %s", err)
		}

		rootCerts = append(rootCerts, cert)
		pemBlock, rawCertBundle = pem.Decode(rawCertBundle)
	}

	if len(rootCerts) == 0 {
		return nil, errors.New("token auth requires at least one token signing root certificate")
	}

	rootPool := x509.NewCertPool()
	trustedKeys := make(map[string]libtrust.PublicKey, len(rootCerts))
	for _, rootCert := range rootCerts {
		rootPool.AddCert(rootCert)
		pubKey, err := libtrust.FromCryptoPublicKey(crypto.PublicKey(rootCert.PublicKey))
		if err != nil {
			return nil, fmt.Errorf("unable to get public key from token auth root certificate: %s", err)
		}

		trustedKeys[pubKey.KeyID()] = pubKey
	}

	return &authStrategy{
		realm:       config.realm,
		issuer:      config.issuer,
		service:     config.service,
		rootCerts:   rootPool,
		trustedKeys: trustedKeys,
	}, nil
}

func getAccessSet(c *RegistryClaims) accessSet {
	accessSet := make(accessSet, len(c.Access))
	return accessSet
}

func (ac *authStrategy) Authorized(ctx context.Context, accessItems ...auth.Access) (context.Context, error) {
	challenge := &authChallenge{
		realm:     ac.realm,
		service:   ac.service,
		accessSet: newAccessSet(accessItems...),
	}

	req, err := context.GetRequest(ctx)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(req.Header.Get("Authorization"), " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		challenge.err = ErrTokenRequired
		return nil, challenge
	}

	rawToken := parts[1]
	claims := &RegistryClaims{}
	token, err := jwt.ParseWithClaims(rawToken, *claims, func(token *jwt.Token) (interface{}, error) {
		/*if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}*/

		return nil, nil
	})

	if err != nil {
		challenge.err = err
		return nil, challenge
	}

	verifyOpts := VerifyOptions{
		TrustedIssuers:    []string{ac.issuer},
		AcceptedAudiences: []string{ac.service},
		Roots:             ac.rootCerts,
		TrustedKeys:       ac.trustedKeys,
	}

	if err = verify(token, *claims, verifyOpts); err != nil {
		challenge.err = err
		return nil, challenge
	}

	accessSet := getAccessSet(claims)
	for _, access := range accessItems {
		if !accessSet.contains(access) {
			challenge.err = ErrInsufficientScope
			return nil, challenge
		}
	}

	return auth.WithUser(ctx, auth.UserInfo{Name: claims.Subject}), nil
}

func verify(token *jwt.Token, claims RegistryClaims, verifyOptions VerifyOptions) error {
	if !contains(verifyOptions.TrustedIssuers, claims.Issuer) {
		log.Errorf("token from untrusted issuer: %q", claims.Issuer)
		return ErrInvalidToken
	}

	if !contains(verifyOptions.AcceptedAudiences, claims.Audience) {
		log.Errorf("token intended for another audience: %q", claims.Audience)
		return ErrInvalidToken
	}

	currentUnixTime := time.Now().Unix()
	if !(claims.NotBefore <= currentUnixTime && currentUnixTime <= claims.ExpiresAt) {
		log.Errorf("token not to be used before %d or after %d - currently %d", claims.NotBefore, claims.ExpiresAt, currentUnixTime)
		return ErrInvalidToken
	}

	if len(token.Signature) == 0 {
		log.Errorf("token has no signature")
		return ErrInvalidToken
	}

	signingKey, err := verifySigningKey(token, verifyOptions)
	if err != nil {
		log.Error(err)
		return ErrInvalidToken
	}

	if err = signingKey.Verify(strings.NewReader(token.Raw), token.Method.Alg(), []byte(token.Signature)); err != nil {
		log.Errorf("unable to verify token signature: %s", err)
		return ErrInvalidToken
	}

	return nil
}

func verifySigningKey(token *jwt.Token, verifyOptions VerifyOptions) (libtrust.PublicKey, error) {
	var err error
	var signingKey libtrust.PublicKey
	x5c, _ := token.Header["x5c"].([]string)
	rawJWK, _ := token.Header["jwk"].(*json.RawMessage)
	keyID, _ := token.Header["kid"].(string)

	switch {
	case len(x5c) > 0:
		signingKey, err = parseAndVerifyCertChain(x5c, verifyOptions.Roots)
	case rawJWK != nil:
		signingKey, err = parseAndVerifyRawJWK(rawJWK, verifyOptions)
	case len(keyID) > 0:
		signingKey = verifyOptions.TrustedKeys[keyID]
		if signingKey == nil {
			err = fmt.Errorf("token signed by untrusted key with ID: %q", keyID)
		}

	default:
		err = errors.New("unable to get token signing key")
	}

	return signingKey, err
}

func parseAndVerifyCertChain(x5c []string, roots *x509.CertPool) (libtrust.PublicKey, error) {
	if len(x5c) == 0 {
		return nil, errors.New("empty x509 certificate chain")
	}

	leafCertDer, err := base64.StdEncoding.DecodeString(x5c[0])
	if err != nil {
		return nil, fmt.Errorf("unable to decode leaf certificate: %s", err)
	}

	leafCert, err := x509.ParseCertificate(leafCertDer)
	if err != nil {
		return nil, fmt.Errorf("unable to parse leaf certificate: %s", err)
	}

	intermediates := x509.NewCertPool()
	for i := 1; i < len(x5c); i++ {
		intermediateCertDer, err := base64.StdEncoding.DecodeString(x5c[i])
		if err != nil {
			return nil, fmt.Errorf("unabel to decode intermediate certificate: %s", err)
		}

		intermediateCert, err := x509.ParseCertificate(intermediateCertDer)
		if err != nil {
			return nil, fmt.Errorf("unable to parse intermediate certificate: %s", err)
		}

		intermediates.AddCert(intermediateCert)
	}

	verifyOptions := x509.VerifyOptions{
		Intermediates: intermediates,
		Roots:         roots,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	// TODO: check for revocations
	if _, err := leafCert.Verify(verifyOptions); err != nil {
		return nil, fmt.Errorf("unable to verify certificate chain: %s", err)
	}

	leafCryptoKey, ok := leafCert.PublicKey.(crypto.PublicKey)
	if !ok {
		return nil, errors.New("unable to get leaf cert public key value")
	}

	leafKey, err := libtrust.FromCryptoPublicKey(leafCryptoKey)
	if err != nil {
		return nil, fmt.Errorf("unable to make libtrust public key from leaf certificate: %s", err)
	}

	return leafKey, nil
}

func parseAndVerifyRawJWK(rawJWK *json.RawMessage, verifyOptions VerifyOptions) (libtrust.PublicKey, error) {
	pubKey, err := libtrust.UnmarshalPublicKeyJWK([]byte(*rawJWK))
	if err != nil {
		return nil, fmt.Errorf("unable to decode raw JWK value: %s", err)
	}

	x5cVal, ok := pubKey.GetExtendedField("x5c").([]interface{})
	if !ok {
		if _, trusted := verifyOptions.TrustedKeys[pubKey.KeyID()]; !trusted {
			return nil, errors.New("untrusted JWK with no certificate chain")
		}

		return pubKey, nil
	}

	x5c := make([]string, len(x5cVal))
	for i, val := range x5cVal {
		certString, ok := val.(string)
		if !ok || len(certString) == 0 {
			return nil, errors.New("malformed certificate chain")
		}

		x5c[i] = certString
	}

	leafKey, err := parseAndVerifyCertChain(x5c, verifyOptions.Roots)
	if err != nil {
		return nil, fmt.Errorf("could not verify JWK certificate chain: %s", err)
	}

	if pubKey.KeyID() != leafKey.KeyID() {
		return nil, errors.New("leaf certificate public key ID does not match JWK key ID")
	}

	return pubKey, nil
}

type ResourceActions struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

type RegistryClaims struct {
	Audience  string `json:"aud,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	Id        string `json:"jti,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	Issuer    string `json:"iss,omitempty"`
	NotBefore int64  `json:"nbf,omitempty"`
	Subject   string `json:"sub,omitempty"`

	// Private claims
	Access []*ResourceActions `json:"access,omitempty"`
}

func (c RegistryClaims) Valid() error {
	// TODO
	return nil
}

func (c *RegistryClaims) accessSet() accessSet {
	accessSet := make(accessSet, len(c.Access))
	for _, resourceActions := range c.Access {
		resource := auth.Resource{
			Type: resourceActions.Type,
			Name: resourceActions.Name,
		}

		set, exists := accessSet[resource]
		if !exists {
			set = newActionSet()
			accessSet[resource] = set
		}

		for _, action := range resourceActions.Actions {
			set.add(action)
		}
	}

	return accessSet
}

func init() {
	auth.Register("token", auth.StrategyFactory(newAuthStrategy))
}
