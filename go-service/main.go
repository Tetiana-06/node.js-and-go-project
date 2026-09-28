// використовує стандартний net/http із підтримкою нового роутингу Go 1.22+ METHOD /path
package main

// Блок імпорту стандартних пакетів без використання зовнішніх сторонніх бібліотек.
import (
	"encoding/json" // Серіалізація та десеріалізація JSON
	"fmt"           // Форматоване виведення в консоль
	"log"           // Логування критичних помилок
	"net/http"      // Стандартний вебсервер і клієнт Go
	"strconv"       // Конвертація рядків у числа (ASCII to Integer)
	"sync"          // Примітиви синхронізації (м'ютекси)
	"time"          // Робота з часом
)

// Допоміжна функція важкого обчислення
func heavyComputation() int64 {
	var count int64 = 0
	for i := int64(0); i < 2e9; i++ {
		count++
	}
	return count
}

// 1. I/O ендпойнт (асинхронний сон горутини)
func ioHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(200 * time.Millisecond)
	writeJSON(w, http.StatusOK, map[string]string{"message": "I/O operation completed"})
}

// 2. CPU: Послідовне виконання важких обчислень
func cpuSyncHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Виконуємо 4 важких цикли послідовно
	c1 := heavyComputation()
	c2 := heavyComputation()
	c3 := heavyComputation()
	c4 := heavyComputation()

	duration := time.Since(start).Milliseconds()
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":     "sequential",
		"duration": fmt.Sprintf("%dms", duration),
		"result":   c1 + c2 + c3 + c4,
	})
}

// 3. CPU: Паралельне виконання в окремих Goroutines
func cpuConcurrentHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var wg sync.WaitGroup
	results := make([]int64, 4)

	// Запускаємо 4 цикли паралельно в окремих горутинах
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = heavyComputation()
		}(i)
	}

	wg.Wait() // Чекаємо завершення всіх 4 горутин

	duration := time.Since(start).Milliseconds()
	var total int64
	for _, val := range results {
		total += val
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"mode":     "concurrent",
		"duration": fmt.Sprintf("%dms", duration),
		"result":   total,
	})
}

// Оголошення структури Task. Анотації у зворотних лапках (struct tags, наприклад `json:"id"`)
// вказують пакету json, у які саме ключі серіалізувати ці поля.
type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// Оголошення глобальних змінних: динамічний слайс tasks, лічильник nextID та mu типу sync.RWMutex для безпечного
// керування конкурентним доступом кількох горутин до слайса.
var (
	tasks  = []Task{{ID: 1, Title: "Learn Cloud Native", Completed: false}}
	nextID = 2
	mu     sync.RWMutex
)

// Допоміжна функція для надсилання JSON
// Допоміжна функція: встановлює заголовок типу вмісту application/json, записує статус HTTP
// (WriteHeader) та транслює передану структуру прямо у вихідний потік відповіді w через потоковий енкодер.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// 1. Health check
// Обробник ендпойнта /health. Приймає інтерфейс запису відповіді w і покажчик на вхідний запит r. Повертає мапу зі статусом 200 OK.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "OK",
		"runtime": "Go net/http",
	})
}

// 2. READ ALL
// Обробник отримання всіх завдань:
// mu.RLock() — бере блокування на читання.
// defer mu.RUnlock() — гарантує зняття блокування в момент виходу з функції.
// Повертає зріз tasks зі статусом 200 OK.
func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	writeJSON(w, http.StatusOK, tasks)
}

// 3. READ ONE
// r.PathValue("id") дістає значення з URL {id} (фіча Go 1.22+). Функція strconv.Atoi пробує перетворити його на число.
//
//	Якщо користувач передав не число (наприклад, /tasks/abc), повертається статус 400 Bad Request.
func getTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	for _, task := range tasks {
		if task.ID == id {
			writeJSON(w, http.StatusOK, task)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Task not found"})
}

// 4. CREATE
// Оголошуємо локальну анонімну структуру тільки для парсингу вхідного тіла запиту (поле ID тут не потрібне, оскільки воно генерується сервером).
func createTaskHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Title is required and valid JSON expected"})
		return
	}
	// mu.Lock() — накладаємо повне ексклюзивне блокування (інші горутини не можуть ані читати, ані писати в цей момент).
	mu.Lock()
	newTask := Task{
		ID:        nextID,
		Title:     input.Title,
		Completed: input.Completed,
	}
	nextID++
	tasks = append(tasks, newTask)
	//mu.Unlock() — знімаємо блокування.
	mu.Unlock()

	writeJSON(w, http.StatusCreated, newTask)
}

// 5. UPDATE
func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
		return
	}

	var input struct {
		Title     *string `json:"title"`
		Completed *bool   `json:"completed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for i, task := range tasks {
		if task.ID == id {
			if input.Title != nil {
				tasks[i].Title = *input.Title
			}
			if input.Completed != nil {
				tasks[i].Completed = *input.Completed
			}
			writeJSON(w, http.StatusOK, tasks[i])
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Task not found"})
}

// 6. DELETE
// Видалення в Go реалізується через зрізи: append(tasks[:i], tasks[i+1:]...) бере всі елементи
//
//	до індексу i та об'єднує з усіма елементами після нього, викидаючи i-й елемент.
func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for i, task := range tasks {
		if task.ID == id {
			tasks = append(tasks[:i], tasks[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Task not found"})
}

// новий мультиплексор запитів (маршрутизатор), який зіставляє шлях URL із відповідною функцією-обробником.
func main() {
	mux := http.NewServeMux()

	// Реєструємо обробники для кожного ендпойнта. Формат: METHOD /path.
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /tasks", getTasksHandler)
	mux.HandleFunc("GET /tasks/{id}", getTaskByIDHandler)
	mux.HandleFunc("POST /tasks", createTaskHandler)
	mux.HandleFunc("PUT /tasks/{id}", updateTaskHandler)
	mux.HandleFunc("DELETE /tasks/{id}", deleteTaskHandler)

	mux.HandleFunc("GET /io", ioHandler)
	mux.HandleFunc("GET /cpu-sync", cpuSyncHandler)
	mux.HandleFunc("GET /cpu-concurrent", cpuConcurrentHandler)
	mux.HandleFunc("GET /cpu", cpuConcurrentHandler) // для загального навантажувального тесту

	port := ":8080"
	fmt.Printf("Go service running on http://localhost%s\n", port)
	// http.ListenAndServe прив'язує сервер до порту 8080 і починає приймати HTTP-з'єднання. Якщо порт зайнятий або виникла фатальна помилка,
	//  log.Fatalf виведе її в консоль і завершить програму.
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
