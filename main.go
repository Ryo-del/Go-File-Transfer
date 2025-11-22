package main

import (
	"fmt"
	"html/template" // Добавлено для работы с шаблонами
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const uploadDir = "./uploads"

// Структура для передачи данных в шаблон
type FileInfo struct {
	Name string
	Size string
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	listenAddr := ":" + port

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.Mkdir(uploadDir, 0755)
		if err != nil {
			log.Fatal("Ошибка при создании директории uploads:", err)
		}
	}

	// 1. Обслуживание статических файлов (HTML, CSS)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// 2. Обработчик загрузки файлов
	http.HandleFunc("/upload", uploadHandler)

	// 3. Обработчик списка файлов
	http.HandleFunc("/manager", managerHandler)

	// 4. Обработчик удаления файлов
	http.HandleFunc("/delete/", deleteHandler)

	// 5. Обслуживание загруженных файлов для скачивания (осталось прежним)
	downloadFs := http.FileServer(http.Dir(uploadDir))
	http.Handle("/files/", http.StripPrefix("/files/", downloadFs))

	fmt.Printf("Сервер запущен на http://localhost%s\n", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

// ... (uploadHandler остается прежним) ...

func uploadHandler(w http.ResponseWriter, r *http.Request) {
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

	// Перенаправляем на страницу менеджера после загрузки
	http.Redirect(w, r, "/manager?uploaded="+header.Filename, http.StatusSeeOther)
}

// --- НОВЫЙ ОБРАБОТЧИК: Отображение списка файлов ---
func managerHandler(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		http.Error(w, "Не удалось прочитать директорию загрузок", http.StatusInternalServerError)
		return
	}

	var fileList []FileInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		info, _ := file.Info()
		fileList = append(fileList, FileInfo{
			Name: file.Name(),
			Size: formatFileSize(info.Size()), // Преобразуем размер
		})
	}

	// Загружаем и исполняем шаблон manager.html
	t, err := template.ParseFiles("./static/manager.html")
	if err != nil {
		log.Println("Ошибка парсинга шаблона manager.html:", err)
		http.Error(w, "Ошибка сервера при загрузке шаблона", http.StatusInternalServerError)
		return
	}

	// Передаем список файлов и сообщение о загрузке в шаблон
	data := struct {
		Files    []FileInfo
		Uploaded string
		Deleted  string
	}{
		Files:    fileList,
		Uploaded: r.URL.Query().Get("uploaded"),
		Deleted:  r.URL.Query().Get("deleted"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, data)
}

// --- НОВЫЙ ОБРАБОТЧИК: Удаление файла ---
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Используйте POST для удаления", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем имя файла из URL: /delete/имя_файла
	filename := strings.TrimPrefix(r.URL.Path, "/delete/")
	if filename == "" {
		http.Error(w, "Имя файла не указано", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(uploadDir, filename)

	// Проверяем, что файл существует
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	// Удаляем файл
	err := os.Remove(filePath)
	if err != nil {
		log.Println("Ошибка при удалении файла:", err)
		http.Error(w, "Ошибка сервера при удалении", http.StatusInternalServerError)
		return
	}

	// Перенаправляем обратно в менеджер с сообщением об успехе
	http.Redirect(w, r, "/manager?deleted="+filename, http.StatusSeeOther)
}

// Вспомогательная функция для форматирования размера файла
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ВАЖНО: Если вы используете старую версию Go, замените os.ReadDir на os.Readdir
