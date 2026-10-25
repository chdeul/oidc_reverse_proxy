package auth

import (
	"context"
	"fmt"
	"main/tools"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// Pour cookie handler zitadel ?! random key
var key = []byte("ad46abc90ef1783bcz6f6h2d4def1879")

var client rp.RelyingParty

// Options handler oidc frontal
var urlOptions = []rp.URLParamOpt{
	// rp.WithPromptURLParam("Welcome back!"),
	rp.WithResponseModeURLParam(oidc.ResponseMode(oidc.ResponseModeQuery)),
}

// Start login procedure
func clientLogin(w http.ResponseWriter, r *http.Request) {

	// Avec query path referer
	p := r.URL.Query().Get("redirect")
	theState := func() string {
		// Un simple multiple de 7 pour anti csrf
		alea := rand.Intn(65535*65535/7) * 7
		// plus path initial de redirection final
		return strconv.Itoa(alea) + "~" + p
	}
	// passe Client oidc qui va rediriger vers Server oidc
	ah := rp.AuthURLHandler(theState, client, urlOptions...)
	ah.ServeHTTP(w, r)
}

// login back from oidc provider to client
func clientCallback(w http.ResponseWriter, r *http.Request,
	tokens *oidc.Tokens[*oidc.IDTokenClaims],
	state string, rp rp.RelyingParty, info *oidc.UserInfo) {

	// state : jeton antcsrf , Url initiale demandée
	path := "/" // default redirection
	values := strings.SplitN(state, "~", 2)
	if len(values) != 2 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Verifier anti csrf : multiple de 7
	alea, err := strconv.Atoi(values[0])
	if err != nil || (alea%7) != 0 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// redirection path
	path, err = url.QueryUnescape(values[1])
	if err != nil || len(path) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Mémorise pour logout
	SetOidcLogoutUrl(rp.GetEndSessionEndpoint())
	idtoken := tokens.IDToken

	// Génère un (nouveau) jeton d'uthentification JWT from OIDC Infos
	_, err = JwtCreateAndTransmit(w, info.PreferredUsername, idtoken, info.Claims["Roles"].(string))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Redirige vers la page initiale
	http.Redirect(w, r, path, http.StatusFound)
}

// logout from oidc server
func clientLogout(w http.ResponseWriter, r *http.Request) {

	// le jeton sera supprimé côté client (stateless app)
	JwtRemoveAndTransmit(w)
	w.WriteHeader(http.StatusOK)
}

func OpClient() error {

	// Option oidc client
	cookieHandler := httphelper.NewCookieHandler(key, key, httphelper.WithUnsecure())

	// WithJWTProfile option
	//keyID := "changeit"
	//signingKey := []byte{cert.pem}

	options := []rp.Option{
		// rp.With ....
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(5 * time.Second)),
		rp.WithSigningAlgsFromDiscovery(),
		//rp.WithJWTProfile(rp.SignerFromKeyAndKeyID(signingKey, keyID)),
	}

	// Trusted https certificates
	ctx := LoadTrustedCertFiles(context.Background())

	// Register as oidc client ( call server oidc .well-known/openid-configuration )
	var err error
	client, err = rp.NewRelyingPartyOIDC(ctx, tools.Cfg.Openidc.Issuer+"/",
		tools.Cfg.Openidc.Ident, tools.Cfg.Openidc.Secret,
		tools.Cfg.Openidc.Callback, tools.Cfg.Openidc.Scope, options...)
	if err != nil {
		fmt.Println("Erreur client OIDC", err)
		return nil
	}

	// Register  /frontal/login  (declenche le login sso)
	http.Handle(tools.Cfg.Openidc.LoginPath, http.HandlerFunc(clientLogin))

	// register the frontchannel_logout_uri   /frontal/logout  (sur logout sso)
	if tools.Cfg.Openidc.FrontLogout != "" {
		u, _ := url.Parse(tools.Cfg.Openidc.FrontLogout)
		http.Handle(u.Path+"/", http.HandlerFunc(clientLogout))
	}
	// Register Callback endpoint   /frontal/callback  (sur bon login sso)
	u, _ := url.Parse(tools.Cfg.Openidc.Callback)
	http.HandleFunc(u.Path+"/", rp.CodeExchangeHandler(rp.UserinfoCallback(clientCallback), client))
	return nil
}
