package add_repo

import (
	"encoding/json"
	"io/ioutil"

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

type DTO struct {
	Branch   string `json:"branch"`
	Path     string `json:"path"`
	Ports    [2]int `json:"ports"`
	Reponame string `json:"repo_name"`
}

func HandleAddRepo(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		c.String(400, err.Error())
	}

	var body DTO

	if err := json.Unmarshal(data, &body); err != nil {
		c.String(400, err.Error())
	}

	filecontent, err := ioutil.ReadFile("config.json")
	var config Config

	if err != nil {
		c.String(500, err.Error())
	}

	if err = json.Unmarshal(filecontent, &config); err != nil {
		c.String(500, err.Error())
	}

	if config.Repos == nil {
		config.Repos = make(map[string]RepoConfig)
	}

	config.Repos[body.Reponame] = RepoConfig{
		Branch: body.Branch,
		Path:   body.Path,
		Ports:  body.Ports,
	}

	updated, _ := json.MarshalIndent(config, "", "  ")
	_ = ioutil.WriteFile("config.json", updated, 0644)

	c.JSON(201, gin.H{
		"details": "repo added",
	})
}
