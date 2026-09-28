const express = require('express');
const app = express(); // екземпляр вебдодатка Express, за допомогою якого реєструються маршрути та підключаються проміжні обробники
const PORT = 3000; //мережевий порт TCP, який слухатиме сервіс

// Мідлвар для парсингу JSON у тілі запиту
// глобальний middleware: перехоплює запити з заголовком Content-Type: application/json, парсить тіло і записує результат у req.body
app.use(express.json());

// In-memory сховище
let tasks = [
  { id: 1, title: 'Learn Cloud Native', completed: false }
];
let nextId = 2;

// 1. Health check
// Реєструємо обробник GET-запиту на /health. Завжди повертає HTTP статус 200 OK та об'єкт готовності сервера.
app.get('/health', (req, res) => {
  res.status(200).json({ status: 'OK', runtime: 'Node.js' });
});

// 2. READ ALL (Отримати всі)
// Реєструємо обробник для читання списку завдань. Серіалізує поточний масив tasks у формат JSON та повертає статус 200 OK.
app.get('/tasks', (req, res) => {
  res.status(200).json(tasks);
});

// 3. READ ONE (Отримати одне за ID)
// Оголошуємо ендпойнт із параметром шляху :id (динамічна частина URL)
app.get('/tasks/:id', (req, res) => {
    // Дістаємо значення :id із параметрів запиту (воно приходить рядком string) і перетворюємо його на ціле число в десятковій системі
  const id = parseInt(req.params.id, 10);
  // Шукаємо в масиві перший елемент, у якого поле id збігається з переданим значенням
  const task = tasks.find(t => t.id === id);

  if (!task) {
    return res.status(404).json({ error: 'Task not found' });
  }

  res.status(200).json(task);
});

// 4. CREATE (Створити нове)
// аршрут для створення нового завдання методом POST. Деструктуризуємо поля title та completed із розпарсеного тіла запиту req.body.
app.post('/tasks', (req, res) => {
  const { title, completed } = req.body;

  // Валідація вхідних даних: якщо поле title відсутнє, повертаємо помилку клієнта 400 Bad Request.
  if (!title) {
    return res.status(400).json({ error: 'Title is required' });
  }
// Створюємо новий об'єкт, автоматично призначаємо йому поточний nextId (і після цього збільшуємо лічильник на одиницю),
//  а також дефолтне значення false для поля completed, якщо воно не було передане.
  const newTask = {
    id: nextId++,
    title,
    completed: completed || false
  };

  tasks.push(newTask);
  res.status(201).json(newTask);
});

// 5. UPDATE (Оновити за ID)
// Обробник PUT для оновлення: знаходимо сутність за ID або повертаємо 404, якщо вона не існує.
app.put('/tasks/:id', (req, res) => {
  const id = parseInt(req.params.id, 10);
  const task = tasks.find(t => t.id === id);

  if (!task) {
    return res.status(404).json({ error: 'Task not found' });
  }
// Перевіряємо, які саме поля надійшли в запиті. Оновлюємо властивості існуючого об'єкта за посиланням і 
// повертаємо оновлений об'єкт зі статусом 200 OK.
  const { title, completed } = req.body;
  if (title !== undefined) task.title = title;
  if (completed !== undefined) task.completed = completed;

  res.status(200).json(task);
});

// 6. DELETE (Видалити за ID)
// Обробник DELETE: шукаємо індекс елемента в масиві за його ідентифікатором.
app.delete('/tasks/:id', (req, res) => {
  const id = parseInt(req.params.id, 10);
  const index = tasks.findIndex(t => t.id === id);
// Якщо індекс дорівнює -1 (елемент відсутній), віддаємо статус 404 Not Found.
  if (index === -1) {
    return res.status(404).json({ error: 'Task not found' });
  }
// Методом splice видаляємо 1 елемент за знайденим індексом. Повертаємо 204 No Content без тіла відповіді 
// (ознака успішного видалення в REST API).
  tasks.splice(index, 1);
  res.status(204).send();
});


// ендпойнти для Лабораторної 2

// 1. Асинхронний I/O ендпойнт (імітуємо затримку I/O у 200мс) ітітація реального життя, ніби ми робили запит до бази даних або зовнішнього API,
//  який займає певний час на виконання.
app.get('/io', async (req, res) => {
  await new Promise(resolve => setTimeout(resolve, 200));
  res.status(200).json({ message: 'I/O operation completed' });
});

// 2. Важка синхронна CPU операція (блокує Event Loop)
// симуляція важких розрахунків, щоб змусити ядро процесора завантажитися на всі 100% суто математикою
app.get('/cpu', (req, res) => {
  let count = 0;
  // Цикл на 5 мільярдів ітерацій для відчутного блокування
  for (let i = 0; i < 5e9; i++) {
    count++;
  }
  res.status(200).json({ message: 'CPU bound task completed', count });
});


// Запускаємо прослуховування вказаного порту TCP і після готовності виводимо в консоль інформаційне повідомлення.
app.listen(PORT, () => {
  console.log(`Node.js service running on http://localhost:${PORT}`);
});