package rproxy

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Requirements for tests
// - Launch proxy  https://localhost:8443
// - with config in sources ( proxy.xml, certif, js,  resources )
// - launch mantis tools http://localhost:8086/mantis
// - with user administrator, password chdeul
// - do not launch  oidc provider  https://localhost:8443/oidc

// All tests
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func call(t *testing.T, url string, code int) {

	timeout := 10 * time.Second // Timeout par requête
	client := &http.Client{Timeout: timeout}
	req, e1 := http.NewRequest("GET", url, nil)
	if e1 != nil {
		t.Fatal("Error NewRequest", e1)
	}
	// Appel
	resp, e2 := client.Do(req)
	if e2 != nil {
		t.Error("Error Request", e2)
	}
	defer resp.Body.Close()
	if resp.StatusCode != code {
		t.Error("Error Request Status", resp.StatusCode, url)
	}
	if resp.ContentLength > 0 {
		var buf []byte
		nb, e3 := resp.Body.Read(buf)

		if e3 != nil {
			t.Error("Error Response read ", e3)
		}
		t.Log("Body of nb bytes", nb)
	}
}

func TestHealth(t *testing.T) {
	call(t, "https://localhost:8443/healthz/live", http.StatusOK)
	call(t, "https://localhost:8443/healthz/ready", http.StatusOK)
}

func TestResources(t *testing.T) {
	call(t, "https://localhost:8443/resources/toolsbox.jpeg", http.StatusOK)
	call(t, "https://localhost:8443/resources/unknown.css", http.StatusNotFound)
}

func TestProxy(t *testing.T) {
	call(t, "https://localhost:8443/mantis/images/mantis_logo.png", http.StatusOK)
	call(t, "http://localhost:8086/mantis/images/mantis_logo.png", http.StatusOK)
	call(t, "https://localhost:8443/mant/images/mantis_logo.png", http.StatusNotFound)
	call(t, "https://localhost:8443/mantisse/images/mantis_logo.png", http.StatusNotFound)
	call(t, "https://localhost:8443/MAntis/", http.StatusNotFound)
	call(t, "https://localhost:8443/mantis/api/soap/mantisconnect.php?wsdl", http.StatusBadGateway)
	call(t, "https://localhost:8443/mantis", http.StatusNotFound)
	call(t, "https://localhost:8443/mantis/", http.StatusOK)
}

func TestSsoAuth(t *testing.T) {
	call(t, "https://localhost:8443/mantis/index.php", http.StatusNotFound)
	// todo if oidc provider running and configured    http.StatusOK
}

func TestIpFilter(t *testing.T) {
	call(t, "https://localhost:8443/mantis/login_password_page.php", http.StatusForbidden)
	call(t, "https://localhost:8443/mantis/images/favicon.ico", http.StatusOK)
}

func TestPhfile(t *testing.T) {
	call(t, "https://localhost:8443/mantis/mantis_offline.php.sample", http.StatusOK)
	call(t, "https://localhost:8443/mantis/mantis_offline.php", http.StatusOK)
	call(t, "https://localhost:8443/mantis/mantis_offline", http.StatusNotFound)
}

func TestExecJs(t *testing.T) {
	call(t, "https://localhost:8443/mantis/dummy/in_out.html", http.StatusOK)
	call(t, "https://localhost:8443/mantis/dummy/rest_call.html", http.StatusOK)
	call(t, "https://localhost:8443/mantis/dummy/ldap_call.html", http.StatusOK)
	call(t, "https://localhost:8443/mantis/login_page.php", http.StatusOK)
	call(t, "https://localhost:8443/mantis/dummy/error.html", http.StatusInternalServerError)
}

/* Test de charge : test --> proxy --> API mantis */
/* Sur le même PC : supporte au moins 50 utilisateurs simultanés sollicitant à fond */
/* c'est mantis qui limite en premier */
func TestCharge(t *testing.T) {

	// Appel API Rest Mantis
	url := "https://localhost:8443/mantis/api/rest/users/username/administrator"
	token := "VARUcfBRltaH14PmwdVZ_AMd-87TLQqz"

	timeout := 10 * time.Second // Timeout par requête
	client := &http.Client{Timeout: timeout}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", token)

	// Charge
	concurrency := 50 // Nombre de goroutines simultanées
	requests := 1000  // Nombre total de requêtes

	var (
		success  int64
		failures int64
		wg       sync.WaitGroup
	)

	// Canal de travail : distribue les "tickets" aux goroutines
	jobs := make(chan int, requests)
	for i := range requests {
		jobs <- i
	}
	close(jobs)

	start := time.Now()

	// Lancer les workers
	for range concurrency {
		wg.Go(func() {
			for range jobs {

				// Appel
				resp, err := client.Do(req)
				if err != nil {
					atomic.AddInt64(&failures, 1)
					continue
				}
				// vider le body pour réutiliser la connexion
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				if resp.StatusCode >= 200 && resp.StatusCode < 400 {
					atomic.AddInt64(&success, 1)
				} else {
					// fmt.Println(" erreur code ", resp)
					atomic.AddInt64(&failures, 1)
				}
			}
		})
	}
	// Mouline
	wg.Wait()

	elapsed := time.Since(start)

	// Statistiques
	fmt.Printf("\n=== Résultats ===\n")
	fmt.Printf("URL            : %s\n", url)
	fmt.Printf("Concurrence    : %d goroutines\n", concurrency)
	fmt.Printf("Requêtes       : %d\n", requests)
	fmt.Printf("Durée totale   : %v\n", elapsed)
	fmt.Printf("Succès         : %d\n", success)
	fmt.Printf("Échecs         : %d\n", failures)
	fmt.Printf("Requêtes/sec   : %.2f\n", float64(requests)/elapsed.Seconds())
}
