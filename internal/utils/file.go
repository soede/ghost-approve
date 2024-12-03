package utils

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
)

type FileInfo struct {
	Type     string `json:"type"`
	Size     int    `json:"size"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

var token string
var apiURL = "https://myteam.mail.ru/bot/v1/files/getInfo"

func InitLoader() error {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	token = os.Getenv("VK_BOT_TOKEN")
	return nil
}

// в vk teeams все файлы передаются по ID, для того чтобы получить ссылку нужно выполнять API запросы
// ссылки временные, поэтому эта функция работает почти всегда
func FileUrlByID(fileID string) (string, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	query := req.URL.Query()
	query.Add("token", token)
	query.Add("fileId", fileID)
	req.URL.RawQuery = query.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warnf("Ошибка при закрытии Body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("не удалось получить информацию о файле: код ответа %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("Ошибка чтения тела: %v\n", err)
		return "", err
	}

	var fileInfo FileInfo
	if err := json.Unmarshal(body, &fileInfo); err != nil {
		return "", fmt.Errorf("ошибка разбора JSON: %v", err)
	}

	return fileInfo.URL, nil
}
