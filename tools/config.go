package tools

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

// Structure de données de config
type Config struct {
	Proxy struct {
		Protocol      string
		ListenIp      string
		Port          string
		CertFile      string
		KeyFile       string
		UsedUrl       string
		TokenSecret   string
		AgeCookie     string
		UserHeader    string
		ResourcesPath string
		Trace         bool
	}
	Openidc struct {
		Active      bool
		Issuer      string
		Ident       string
		Secret      string
		LoginPath   string
		Callback    string
		PostLogout  string
		FrontLogout string
		Scope       []string
		AuthMethod  string
		UidField    string
		RootCA      string
	}
	Backends []struct {
		Host string
		Path string
		Url  string
	}
	Locations []struct {
		Path    string
		Auth    bool
		Users   string
		IpList  string
		Content []byte
		ExecJs  string
		PhFile  string
	}
}

// Config globale chargée
var Cfg Config

func LoadProxyConfig(dir string) error {

	// Lecture fichier
	err := LoadAnyConfig(dir+"/proxy.yml", &Cfg)
	if err != nil {
		fmt.Println("Fichier config erroné : ", err)
		return err
	}
	// Preference to env variables
	loadEnvironment()
	return err
}

// Charge une configuration yaml
func LoadAnyConfig(filepath string, ptr any) error {

	data, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Println("Fichier config non trouvé", filepath)
		return err
	}
	if err := yaml.Unmarshal(data, ptr); err != nil {
		fmt.Println("Fichier config erroné : ", err)
		return err
	}
	return err
}

func loadEnvironment() {

	v := os.Getenv("USED_URL")
	if v != "" {
		Cfg.Proxy.UsedUrl = v
	}
	v = os.Getenv("RESOURCES_PATH")
	if v != "" {
		Cfg.Proxy.ResourcesPath = v
	}
	v = os.Getenv("TRACE")
	if v != "" {
		Cfg.Proxy.Trace = (v == "TRUE")
	}
	v = os.Getenv("CERT_FILE")
	if v != "" {
		Cfg.Proxy.CertFile = v
	}
	v = os.Getenv("AGE_COOKIE")
	if v != "" {
		Cfg.Proxy.AgeCookie = v
	}
	v = os.Getenv("USER_HEADER")
	if v != "" {
		Cfg.Proxy.UserHeader = v
	}
	v = os.Getenv("OIDC_ISSUER")
	if v != "" {
		Cfg.Openidc.Issuer = v
	}
	v = os.Getenv("OIDC_IDENT")
	if v != "" {
		Cfg.Openidc.Ident = v
	}
	v = os.Getenv("OIDC_SECRET")
	if v != "" {
		Cfg.Openidc.Secret = v
	}
	v = os.Getenv("OIDC_AUTH_METHOD")
	if v != "" {
		Cfg.Openidc.AuthMethod = v
	}
	v = os.Getenv("OIDC_UID_FIELD")
	if v != "" {
		Cfg.Openidc.UidField = v
	}
	v = os.Getenv("OIDC_ROOT_CA")
	if v != "" {
		Cfg.Openidc.RootCA = v
	}
}
