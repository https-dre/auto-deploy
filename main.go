package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Secret      string                `json:"secret"`
	CurrentPort string                `json:"current_port"`
	Repos       map[string]RepoConfig `json:"repos"`
}

type RepoConfig struct {
	Branch string `json:"branch"`
	Path   string `json:"path"`
	Ports  [2]int `json:"ports"`
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

	r := gin.Default()

	r.POST("/webhook", handleWebhook)

	fmt.Println("Servidor ouvindo na porta", config.CurrentPort)
	if err := r.Run(":" + config.CurrentPort); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}

func handleWebhook(c *gin.Context) {
	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		c.String(http.StatusForbidden, "Sem assinatura")
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		c.String(http.StatusBadRequest, "Erro ao ler corpo da requisição")
		return
	}

	// Verifica assinatura
	if !verifySignature(signature, body, []byte(config.Secret)) {
		c.String(http.StatusForbidden, "Assinatura inválida")
		return
	}

	var payload struct {
		Ref        string `json:"ref"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		c.String(http.StatusBadRequest, "Payload inválido")
		return
	}

	repo := payload.Repository.FullName
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")

	if repoConf, ok := config.Repos[repo]; ok && repoConf.Branch == branch {
		go deploy(repoConf)
	}

	c.Status(http.StatusOK)
}

func verifySignature(signature string, body, secret []byte) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func deploy(repo RepoConfig) {
	fmt.Println("Executando deploy para", repo.Path)

	port1 := strconv.Itoa(repo.Ports[0])
	port2 := strconv.Itoa(repo.Ports[1])

	cmd := exec.Command("bash", "./up-docker.sh", port1, port2)
	cmd.Dir = repo.Path // melhor que "./", para garantir que está no projeto certo

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Erro:", err)
	}
	fmt.Println(string(output))
}
