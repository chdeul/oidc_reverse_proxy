package rproxy

import (
	"fmt"
	"main/auth"
	"main/tools"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"
)

// Var globales : prétraitement invariant
var proxies []*httputil.ReverseProxy
var ips [][]string
var users [][]string

// Decompose une url string
func mustParse(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

// Equivalence de hostname  *.xxx.yy  zzz.xxx.yy
func matchDomain(pattern, domain string) bool {

	// Simple (cas sensitive)
	if !strings.HasPrefix(pattern, "*.") {
		return pattern == domain
	}
	// Wildcard
	suffix := pattern[1:] // sans *
	return strings.HasSuffix(domain, suffix) && len(strings.TrimSuffix(domain, suffix)) > 1
}

// Get IP of input request
func getSourceIP(r *http.Request) string {

	// 1. X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// 2. X-Real-IP
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	// 3. RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}
	return r.RemoteAddr
}

// Ip matching cidr expression
func ipInRange(ipStr, cidr string) bool {

	// remote IP correcte
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	// Exact match  (no cidr range)
	if ipStr == cidr {
		return true
	}
	// Range  /xx
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Bad cidr range
		return false
	}
	// Check range
	return ipNet.Contains(ip)
}

// Fait ce qui peut être fait avant, une seule fois
func Init() {

	// Liste des backends proxies
	proxies = make([]*httputil.ReverseProxy, len(tools.Cfg.Backends))

	// Chaque Backend possible : dans l'ordre
	for i, tbe := range tools.Cfg.Backends {

		// Le crée
		ube := mustParse(tbe.Url)
		proxy := &httputil.ReverseProxy{
			Rewrite: func(req *httputil.ProxyRequest) {

				// Changement host,protocol -> Out req
				req.Out.URL.Scheme = ube.Scheme
				req.Out.URL.Host = ube.Host

				// Recombinaison du path
				if tbe.Path != ube.Path {
					suf, _ := strings.CutPrefix(req.Out.URL.Path, tbe.Path)
					req.Out.URL.Path = path.Join(ube.Path, suf)
					req.Out.URL.RawPath = req.Out.URL.EscapedPath()
				}
				// Ajout les X-forwarded
				req.SetXForwarded()

				// Ajout Header User Authentifié eventuel X-Forwarded-User
				// => fait dans Filtre location uniquement sur les authentifiés
				//user := req.In.Header.Get(tools.Cfg.Proxy.UserHeader)
				//req.Out.Header.Set(tools.Cfg.Proxy.UserHeader, user)
			},
			// Transport: intercepteur(trace),   // Pour dev trace
		}
		// Erreur retransmission and trace
		proxy.ErrorHandler = errorHandler

		// Trace des requetes eventuelle
		if tools.Cfg.Proxy.Trace {
			proxy.ModifyResponse = func(resp *http.Response) error {
				fmt.Println("--> Backend status", resp.StatusCode)
				return nil
			}
		}
		// Le mémorse
		proxies[i] = proxy
	}

	// Liste des listes d'ip splitées
	ips = make([][]string, len(tools.Cfg.Locations))

	// Liste des listes d'users splités
	users = make([][]string, len(tools.Cfg.Locations))

	// Pour chaque Location possible : dans l'ordre
	for i, loc := range tools.Cfg.Locations {
		lu := strings.Split(loc.Users, "|")
		var lu2 []string
		for _, u := range lu {
			w := strings.TrimSpace(u)
			if w != "" {
				lu2 = append(lu2, w)
			}
		}
		users[i] = lu2
		li := strings.Split(loc.IpList, "|")
		var li2 []string
		for _, i := range li {
			w := strings.TrimSpace(i)
			if w != "" {
				li2 = append(li2, w)
			}
		}
		ips[i] = li2
	}

	// Point d'entree, toutes sortes d'url et virtual host
	// Fait le job dynamique à chaque requete (chaque goroutine)
	http.HandleFunc("/", serveProxy)
}

// Trouve la bonne route
func serveProxy(w http.ResponseWriter, r *http.Request) {

	// Filtrage sur ip,user,auth
	status, data, code := filtreLocation(r)
	if status == http.StatusOK {

		// Trouve le bon backend
		i := filtreBackend(r.Host, r.URL.Path)
		if i < 0 {
			// Pas trouvé
			w.WriteHeader(http.StatusNotFound)
			return
		}
		proxy := proxies[i]
		if proxy == nil {
			// Pas trouvé
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Transmet la requette au back end et recoit la reponse
		proxy.ServeHTTP(w, r)
		return
	}
	// Requiert une auth préalable
	if status == http.StatusNonAuthoritativeInfo {

		// Si SSO configuré
		if tools.Cfg.Openidc.Active {

			// Redirigine en interne vers SSO
			redir := "?redirect=" + r.URL.EscapedPath()
			http.Redirect(w, r, tools.Cfg.Openidc.LoginPath+redir, http.StatusFound)

		} else {
			// Non autorisé
			w.WriteHeader(http.StatusForbidden)
		}
		return
	}
	// Intercepté et renvoie un contenu calculé/patché
	if status == http.StatusAlreadyReported {

		// Custom response data
		if len(data) > 0 {

			// data non nulles
			// Cas standards seulement ( json ou formdata )
			if data[0] == '{' || data[0] == '[' {
				w.Header().Add("Content-Type", "application/json")
				w.Write(data)
			} else if r.Method == http.MethodPost {
				w.Header().Add("Content-Type", "application/x-www-form-urlencoded")
				w.Write(data)
			} else { // texte, html
				w.Write(data)
			}
		}
		// Custom http status code
		if code >= http.StatusOK {
			w.WriteHeader(code)
		} else if len(data) == 0 {
			w.WriteHeader(http.StatusOK)
		}
		return // http.StatusOK
	}
	// Non conforme aux filtres : status d'erreur
	w.WriteHeader(status)
}

// Trouve vers le bon backend
func filtreBackend(host, upath string) int {

	// Chemin demandé: non vide
	path := upath
	if path == "" {
		path = "/"
	}
	// Backends possibles : dans l'ordre
	for i, tbe := range tools.Cfg.Backends {
		// Si non virtual host ou  host matche
		if tbe.Host == "" || matchDomain(tbe.Host, host) {
			// Si contextroot matche
			if strings.HasPrefix(path, tbe.Path) {
				// Trouvé
				return i
			}
		}
	}
	// Non trouvé
	return -1
}

func filtreLocation(r *http.Request) (int, []byte, int) {

	// Ip source (si besoin)
	ip := ""

	// Trace des requetes eventuelle
	if tools.Cfg.Proxy.Trace {
		ip = getSourceIP(r)
		fmt.Println(ip, r.Host, r.Method, r.URL.Path, r.URL.RawQuery)
	}

	// Chemin demandé: non vide
	path := r.URL.Path
	if path == "" {
		path = "/"
	}

	// Location posssibles : dans l'ordre
	for i, loc := range tools.Cfg.Locations {

		// Si Matche
		if strings.HasPrefix(path, loc.Path) {

			// Contrainte authentifié
			if loc.Auth {

				// La requete est-elle authentifiée (by JWT token)
				user := getAuthUser(r)

				// Echec ou pas d'authentification
				if user == "" {
					// Efface le header éventuel auth reçu X-Forwarded-User
					r.Header.Del(tools.Cfg.Proxy.UserHeader)

					// Demande une redirection vers sso
					return http.StatusNonAuthoritativeInfo, nil, 0
				}
				// Transmet au backend l'utilisateur logué X-Forwarded-User
				r.Header.Set(tools.Cfg.Proxy.UserHeader, user)

				// Contrainte user seulements
				if loc.Users != "" {

					// Liste des users possibles
					lu := users[i]

					// Utilisateur non autorisé ?
					if !slices.Contains(lu, user) {
						return http.StatusForbidden, nil, 0
					}
				}
			}
			// Contrainte ip source
			if loc.IpList != "" {

				// Ip source de la requete (si pas déjà)
				if ip == "" {
					ip = getSourceIP(r)
				}
				// Liste des cidr possibles
				li := ips[i]

				// Cherche si matche
				found := false
				for _, cidr := range li {
					if cidr != "" && ipInRange(ip, cidr) {
						found = true
						break
					}
				}
				// Ip Non autorisée ?
				if !found {
					return http.StatusForbidden, nil, 0
				}
			}
			// Javascript to execute ( uniquement si pas déjà en cours )
			if loc.ExecJs != "" && r.Header.Get("X-Internal-Call") != "true" {

				// Précharge
				if len(loc.Content) == 0 {
					loc.Content = loadContent(loc.ExecJs)
				}
				// On error : do not retry
				if len(loc.Content) == 0 {
					loc.ExecJs = ""
					return http.StatusNotFound, nil, 0
				}

				// Execute Js file
				data, code := executeJs(loc.Content, r)

				if data != nil || code > 0 {
					// Return custom reponse
					return http.StatusAlreadyReported, data, code
				}

				// Appel normal r, otionnellement patchée par js
				return http.StatusOK, nil, 0

				// Place holder file to return
			} else if loc.PhFile != "" {

				// Précharge
				if len(loc.Content) == 0 {
					loc.Content = loadContent(loc.PhFile)
				}
				// On error : do not retry
				if len(loc.Content) == 0 {
					loc.PhFile = ""
					return http.StatusNotFound, nil, 0
				}
				// Ne transmet pas au backend, retourne content
				return http.StatusAlreadyReported, loc.Content, http.StatusOK

				// Placeholder direct content to return
			} else if len(loc.Content) > 0 {

				// Ne transmet pas au backend, retourne content
				return http.StatusAlreadyReported, loc.Content, http.StatusOK
			}
		}
	}
	// Contraintes passées
	return http.StatusOK, nil, 0
}

// Charge un fichier
func loadContent(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Fichier non trouvé ! :v" + filename)
		return nil
	}
	return data
}

// Regarde si l'appelant s'était déjà authentifié
func getAuthUser(r *http.Request) string {

	// Extract cookie JWT
	token := auth.ExtractJwt(r)
	if token == "" {
		return "" // No one
	}
	// Extract user
	user, _, _ := auth.ParseToken(token)

	// Logged user ( or "" if expired or invalid )
	return user
}

// En cas d'erreur
func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusBadGateway)
}

/*  For Dev : see call of trace

func trace(req *http.Request) (*http.Response, error) {
	fmt.Println("→ Backend URL: %s\n", req.URL.String())
	return http.DefaultTransport.RoundTrip(req)
}

type intercepteur func(*http.Request) (*http.Response, error)

func (f intercepteur) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
*/
