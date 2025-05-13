package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type Config struct {
	Secret string                `json:"secret"`
	Repos  map[string]RepoConfig `json:"repos"`
}

type RepoConfig struct {
	Branch string `json:"branch"`
	Path   string `json:"path"`
	Script string `json:"script"`
}

var config Config

func main() {
	// Carrega config.json
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal("Erro ao ler config.json:", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatal("Erro ao parsear config:", err)
	}

	http.HandleFunc("/webhook", handleWebhook)
	fmt.Println("Servidor ouvindo na porta 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "Sem assinatura", http.StatusForbidden)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	// Verifica assinatura
	if !verifySignature(signature, body, []byte(config.Secret)) {
		http.Error(w, "Assinatura inválida", http.StatusForbidden)
		return
	}

	var payload struct {
		Ref        string `json:"ref"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	repo := payload.Repository.FullName
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	if repoConf, ok := config.Repos[repo]; ok && repoConf.Branch == branch {
		go deploy(repoConf)
	}

	w.WriteHeader(http.StatusOK)
}

func verifySignature(signature string, body, secret []byte) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func deploy(repo RepoConfig) {
	fmt.Println("Executando deploy para", repo.Path)
	cmd := exec.Command("bash", repo.Script)
	cmd.Dir = repo.Path
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Erro:", err)
	}
	fmt.Println(string(output))
}
