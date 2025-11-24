// main.go (veya API'nizin ana dosyası)

package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Ping/Health Check endpoint'i ekleme
	http.HandleFunc("/api/ping", handlePing)

	// Diğer tüm route'larınız burada tanımlanır.
	// http.HandleFunc("/api/movies", handleMovies)

	fmt.Println("Server is running on port 8080...")
	http.ListenAndServe(":8080", nil)
}

// Yeni handler fonksiyonu
func handlePing(w http.ResponseWriter, r *http.Request) {
	// Uygulamanın çalışıp çalışmadığını belirten basit bir yanıt döner
	w.WriteHeader(http.StatusOK) // HTTP 200 OK
	w.Write([]byte("pong"))      // Yanıt gövdesi
}
