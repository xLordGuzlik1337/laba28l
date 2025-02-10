package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	oauth2Config = oauth2.Config{
		ClientID:     "Ov23lidNTGpFNjCFoj9F", 
		ClientSecret: "4496db3ff63c1f3f6505facef6f9338e95776277", 
		RedirectURL:  "http://localhost:8025/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
	// Создаём хранилище сессий
	store = sessions.NewCookieStore([]byte("random-secret-key"))
)

func main() {
	http.HandleFunc("/", handleMain)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/profile", handleProfile)
	http.HandleFunc("/logout", handleLogout) 

	log.Println("Сервер запущен на http://localhost:8025")
	log.Fatal(http.ListenAndServe(":8025", nil))
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	tmpl := `<html>
		<body>
			<h1>Авторизация через GitHub</h1>
			<a href="/login">Войти через GitHub</a>
		</body>
	</html>`
	fmt.Fprint(w, tmpl)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	url := oauth2Config.AuthCodeURL("", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	token, err := oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	client := oauth2Config.Client(r.Context(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		http.Error(w, "Failed to parse user info", http.StatusInternalServerError)
		return
	}

	
	session, _ := store.Get(r, "session-name")
	session.Values["token"] = token.AccessToken
	session.Values["username"] = user.Login
	session.Save(r, w)

	http.Redirect(w, r, "/profile", http.StatusFound)
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	token, ok := session.Values["token"].(string)
	if !ok || token == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	client := oauth2Config.Client(r.Context(), &oauth2.Token{AccessToken: token})
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		http.Error(w, "Failed to parse user info", http.StatusInternalServerError)
		return
	}

	tmpl := `<html>
		<body>
			<h1>Профиль</h1>
			<p>Username: {{.Login}}</p>
			<a href="/logout">Выйти</a>
		</body>
	</html>`
	t, _ := template.New("profile").Parse(tmpl)
	t.Execute(w, user)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Options.MaxAge = -1 
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}
