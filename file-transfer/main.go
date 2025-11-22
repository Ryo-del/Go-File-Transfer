package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Используем ту же директорию, но помним о временном хранении в облаке
const uploadDir = "./uploads"

func main() {
	// --- ИЗМЕНЕНИЕ: Динамическое определение порта ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Дефолт для локального запуска
	}
	listenAddr := ":" + port
	// --- Конец изменения ---

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.Mkdir(uploadDir, 0755)
		if err != nil {
			log.Fatal("Ошибка при создании директории uploads:", err)
		}
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/upload", uploadHandler)

	downloadFs := http.FileServer(http.Dir(uploadDir))
	http.Handle("/files/", http.StripPrefix("/files/", downloadFs))

	fmt.Printf("Сервер запущен на http://localhost%s\n", listenAddr)
	fmt.Printf("Загрузки будут сохранены в %s\n", uploadDir)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// ... (Остальная часть uploadHandler остается без изменений)
	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(100 << 20)

	file, header, err := r.FormFile("uploadFile")
	if err != nil {
		http.Error(w, "Ошибка при получении файла: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	newFileName := filepath.Join(uploadDir, header.Filename)
	newFile, err := os.Create(newFileName)
	if err != nil {
		http.Error(w, "Не удалось создать файл на сервере", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, file)
	if err != nil {
		http.Error(w, "Ошибка при записи файла на диск", http.StatusInternalServerError)
		return
	}

	downloadURL := fmt.Sprintf("/files/%s", header.Filename)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Здесь можно использовать более сложный шаблон для красивого вывода
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Успешно!</title>
			<link rel="stylesheet" href="/style.css">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body>
			<div class="container">
				<h2>Файл успешно загружен! ✅</h2>
				<p>Теперь его можно скачать с любого устройства:</p>
				<a class="button" href="%s" download="%s">Скачать %s</a>
				<br><br>
				<a href="/">Загрузить еще один файл</a>
			</div>
		</body>
		</html>
	`, downloadURL, header.Filename, header.Filename)
}
