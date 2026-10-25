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
