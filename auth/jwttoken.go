package auth

import (
	"errors"
	"main/tools"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

const (
	// Config Private key, the default
	secretkey = "ma_Clef!Secrète."
)

var SsoLogoutUrl string

func ExtractJwt(r *http.Request) string {

	// Jeton d'auth présent Dans Header ou Cookie
	jwtToken := r.Header.Get("Authorization")
	if jwtToken == "" {
		cook, err := r.Cookie("JWTAgremToken")
		if err == nil {
			jwtToken = cook.Value
		}
	}
	if jwtToken != "" {
		jwtToken = strings.Replace(jwtToken, "Bearer ", "", 1)
	}
	return jwtToken
}

func CreateToken(ident, profil, oidcToken string) (string, error) {

	secret := tools.Cfg.Proxy.TokenSecret
	if secret == "" {
		secret = secretkey
	}

	// Creation with expiration date, no refresh token
	maxAge := 60 * 60 * 10                                            // seconds => 10h => 1 jour ouvré
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{ // HMAC-SHA 256
		"identifiant": ident,                                                      // Username
		"profil":      profil,                                                     // Droits
		"ssotoken":    oidcToken,                                                  // empty  or  sso token
		"exp":         time.Now().Add(time.Duration(maxAge) * time.Second).Unix(), // Expiration
	})
	// Sign and get the complete encoded token as a string using the secret
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenString string) (ident, profil, oidcToken string) {

	secret := tools.Cfg.Proxy.TokenSecret
	if secret == "" {
		secret = secretkey
	}

	// Decode
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	// Si mauvais
	if err != nil {
		// Utilisateur vide
		return "", "", ""
	}

	// Extrait les données
	claims, ok := token.Claims.(jwt.MapClaims)

	// Si Invalid ou Expiré
	if !ok || !token.Valid {
		// Utilisateur vide
		return "", "", ""
	}

	// Return data ( identifiant et profil de droits )
	return claims["identifiant"].(string), claims["profil"].(string), claims["ssotoken"].(string)
}

func JwtCreateAndTransmit(w http.ResponseWriter, identifiant string, idOidc string, profil string) (string, error) {

	// Génère un (nouveau) jeton
	token, err := CreateToken(identifiant, profil, idOidc)
	if err != nil {
		return "", err
	}
	// Url extérieure des outils
	parsed, err := url.Parse(tools.Cfg.Proxy.UsedUrl)
	if err != nil {
		return "", err
	}
	isHttps := (parsed.Scheme == "https")
	domaine := parsed.Hostname()

	// Un cookie  valable pour tous les appli controlés : no context  /
	cook := http.Cookie{Name: "JWTAgremToken", Value: token, Domain: domaine, Path: "/", HttpOnly: false, Secure: isHttps}

	// Durée de session
	age := tools.Cfg.Proxy.AgeCookie
	if idOidc != "" {
		age = tools.Cfg.Proxy.AgeCookie
	}
	if age != "session" {
		cook.MaxAge, _ = strconv.Atoi(tools.Cfg.Proxy.AgeCookie)
	}

	// Retourne le jeton JWT D'authentification dans un cookie
	http.SetCookie(w, &cook)

	// Login ok,
	return token, err
}

func JwtRemoveAndTransmit(w http.ResponseWriter) {

	// Url extérieure des outils
	parsed, err := url.Parse(tools.Cfg.Proxy.UsedUrl)
	if err != nil {
		return
	}
	isHttps := (parsed.Scheme == "https")
	domaine := parsed.Hostname()

	// Un cookie  valable pour tous les appli controlés : no context  /  ( pour le proxy)
	cook := http.Cookie{Name: "JWTAgremToken", Value: "", Domain: domaine, Path: "/", HttpOnly: false, Secure: isHttps, MaxAge: -1}

	// Effacer   Token: ""  MaxAge: -1
	http.SetCookie(w, &cook)
}

// For SSO login/logout
func SetOidcLogoutUrl(url string) {
	SsoLogoutUrl = url
}
func GetOidcLogoutUrl() string {
	return SsoLogoutUrl
}
