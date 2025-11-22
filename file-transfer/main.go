package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const uploadDir = "./uploads"

func main() {
	// Создаем директорию для загрузки, если она не существует
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.Mkdir(uploadDir, 0755)
		if err != nil {
			log.Fatal("Ошибка при создании директории uploads:", err)
		}
	}

	// 1. Обслуживание статических файлов (HTML, CSS)
	// http.FileServer будет отдавать содержимое директории static
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// 2. Обработчик загрузки файлов
	http.HandleFunc("/upload", uploadHandler)

	// 3. Обслуживание загруженных файлов для скачивания
	// Файлы будут доступны по /files/{имя_файла}
	downloadFs := http.FileServer(http.Dir(uploadDir))
	http.Handle("/files/", http.StripPrefix("/files/", downloadFs))

	port := ":8080"
	fmt.Printf("Сервер запущен на http://localhost%s\n", port)
	fmt.Printf("Загрузки будут сохранены в %s\n", uploadDir)
	log.Fatal(http.ListenAndServe(port, nil))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Устанавливаем максимальный размер загрузки, например, 100MB
	r.ParseMultipartForm(100 << 20) // 100MB

	file, header, err := r.FormFile("uploadFile")
	if err != nil {
		http.Error(w, "Ошибка при получении файла: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Создаем новый файл на сервере
	newFileName := filepath.Join(uploadDir, header.Filename)
	newFile, err := os.Create(newFileName)
	if err != nil {
		http.Error(w, "Не удалось создать файл на сервере", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	// Копируем загруженный файл в новый файл на сервере
	_, err = io.Copy(newFile, file)
	if err != nil {
		http.Error(w, "Ошибка при записи файла на диск", http.StatusInternalServerError)
		return
	}

	// Перенаправляем пользователя на страницу скачивания (которая по сути является списком файлов)
	// В минималистичном варианте просто выведем ссылку на скачивание.
	// Более сложный вариант может быть списком файлов.
	downloadURL := fmt.Sprintf("/files/%s", header.Filename)

	// Используем простой HTML-ответ для перенаправления/уведомления
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
