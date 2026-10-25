package rproxy

import (
	"bytes"
	"fmt"
	"io"
	"main/tools"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-ldap/ldap/v3"
)

// Execute javascript with an integrated js-runtime
func executeJs(javascript []byte, r *http.Request) (data []byte, code int) {

	// Tracer les erreurs fatales
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Panic execjs récupérée :", r)
			code = http.StatusInternalServerError
			data = nil
		}
	}()

	// Js runtime embed in golang
	vm := goja.New()

	// Set a Go value into JS: remote user authenticated if any
	vm.Set("Remote_User", getAuthUser(r))

	// Calcule l'url frontend et backend, sans query
	beUrl := ""
	path := r.URL.Path
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	feUrl := fmt.Sprintf("%s://%s%s", scheme, r.Host, path)

	i := filtreBackend(r.Host, path)
	if i >= 0 {
		if path == "" {
			path = "/"
		}
		be := tools.Cfg.Backends[i]
		path = strings.TrimPrefix(path, be.Path)
		beUrl = be.Url + path
	}
	vm.Set("Frontend_URL", feUrl)
	vm.Set("Backend_URL", beUrl)
	vm.Set("Query_String", r.URL.RawQuery)

	// Autres données potentiellement interessantes
	vm.Set("Authorization", r.Header.Get("Authorization"))
	vm.Set("Cookie", r.Header.Get("Cookie"))
	vm.Set("Remote_IP", getSourceIP(r))
	vm.Set("Method", r.Method)
	vm.Set("Content_Type", r.Header.Get("Content-Type"))

	// Content-Type
	ct := r.Header.Get("Content-Type")
	isJson := strings.Contains(r.Header.Get("Content-Type"), "application/json")
	isFdata := strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	vm.Set("Content_Type", ct)
	nb := r.ContentLength

	// Json ou form-data body
	if r.Body != nil && nb > 0 && (isJson || isFdata) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		vm.Set("Body", string(body))
	} else {
		vm.Set("Body", "")
	}
	// Functions usable into js
	registerTools(vm)

	// Execute JS using that variables, result map
	result, err := vm.RunString(string(javascript))
	if err != nil {
		fmt.Println("Erreur d'exécution js : ", err)
		return nil, http.StatusInternalServerError
	}
	// Resultat du js : map de patchs requete ou données
	sdata := ""
	code = 0
	ty := result.ExportType().String()
	if strings.HasPrefix(ty, "map") {
		var values map[string]string
		err = vm.ExportTo(result, &values)
		if err != nil {
			fmt.Println("Erreur de retour js : ", err)
			return nil, http.StatusInternalServerError
		}
		// Data ou Patchs de la requete
		sdata, code = rewriteRequest(r, values)
	} else if ty == "string" {
		sdata = result.String()
	} else if strings.HasPrefix(ty, "int") {
		code = int(result.ToInteger())
	}
	if sdata == "" {
		// Requete Patchée et traitement normal
		return nil, code
	}
	// Data directement retournées par le js
	data = []byte(sdata)
	return data, code
}

// Selon resultat js : Modifie la requete initiale éventuellement
func rewriteRequest(r *http.Request, values map[string]string) (string, int) {

	var data string
	var code int = 0
	for param, val := range values {

		// Patchs possibles de requete
		switch param {
		case "Method":
			r.Method = val
		case "Host":
			r.Host = val
		case "Url":
			r.URL, _ = url.Parse(val)
		case "Body":
			r.Body = io.NopCloser(strings.NewReader(val))
			r.ContentLength = int64(len(val))
		// juste des données éventuelle
		case "Code":
			code, _ = strconv.Atoi(val)
		case "Data":
			data = val
		// Header divers
		default:
			r.Header.Set(param, val)
		}
	}
	return data, code // ou rien
}

func registerTools(vm *goja.Runtime) {

	// Function to do in js synchronized Http or Ldap Call
	vm.Set("httpCall",

		// Http func callable from js
		func(method, url, token, body, cookie string) string {

			var ct string
			var buf bytes.Buffer

			// Type de content
			if body != "" {
				// json body
				ct = "application/json"
				buf.ReadFrom(strings.NewReader(body))
				// TODO  xxx=bbb&yyy=aaa   ( form-data )
				ct = "application/x-www-form-urlencoded"
			} else {
				// No body
				ct = "text/html; charset=utf-8"
			}

			// Requete
			req, err := http.NewRequest(method, url, &buf)
			if err != nil {
				return "Request from js error : " + err.Error()
			}
			req.Header.Set("Content-Type", ct)
			req.Header.Set("X-Internal-Call", "true")
			req.Header.Set("Cookie", cookie)
			if token != "" {
				req.Header.Set("Authorization", token)
			}
			// Appel depuis js
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					// Evite le rebouclage sur redirection dans execjs
					req.Header.Set("X-Internal-Call", "true")
					//return http.ErrUseLastResponse  // Ne Traite pas la redirection
					return nil // Traite la redirection
				}}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("REST request error :", err)
				vm.Set("Resp_Status", http.StatusInternalServerError)
				return ""
			}
			// Recup des données ( json, xml, text ...)
			defer resp.Body.Close()
			data, _ := io.ReadAll(resp.Body)

			// Passer en retour au js
			lc := resp.Header.Get("Set-Cookie")
			vm.Set("Set_cookie", lc)
			vm.Set("Status", resp.StatusCode)

			// Renvoie les données au js
			return string(data)
		})

	vm.Set("ldapCall",

		// Ldap search func callable from js
		func(url, bind, pwd, dn, search, attr string) string {

			// Connection open
			l, err := ldap.DialURL(url)
			if l == nil || err != nil {
				return ""
			}
			defer l.Close()

			// Authentified read for get password
			if pwd != "" {
				err = l.Bind(bind, pwd)
			} else {
				err = l.UnauthenticatedBind("")
			}

			// TODO not children only
			// Filter on objets "ou=users,dc=ldap,dc=example,dc=org"
			searchRequest := ldap.NewSearchRequest(dn,
				ldap.ScopeChildren, ldap.NeverDerefAliases, 0, 0, false,
				search, strings.Split(attr, ","), nil)

			// Search for the matching attr
			sr, err := l.Search(searchRequest)
			if err != nil {
				fmt.Println("LDAP request error :", err)
				vm.Set("Status", http.StatusInternalServerError)
				return ""
			}
			// Retourne les valeurs des attributs
			// Dans le cas le plus simple : password=xyz
			var result strings.Builder
			for i, entry := range sr.Entries {
				if i != 0 {
					result.WriteString("\n")
				}
				for j, attr := range entry.Attributes {
					if j != 0 {
						result.WriteString(",")
					}
					result.WriteString(attr.Name)
					result.WriteString("=")
					for k, val := range attr.Values {
						if k != 0 {
							result.WriteString(":")
						}
						result.WriteString(val)
					}
				}
			}
			if result.Len() > 0 {
				vm.Set("Status", http.StatusOK)
			} else {
				vm.Set("Status", http.StatusNoContent)
			}
			return result.String()
		})
}
